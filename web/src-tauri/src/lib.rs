use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::{Aes256Gcm, Nonce};
use base64::engine::general_purpose::STANDARD as BASE64;
use base64::Engine;
use rand::RngCore;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use sha2::{Digest, Sha256};
use std::fs;
use std::net::{SocketAddr, TcpStream};
use std::path::PathBuf;
use std::process::{Child, Command, Stdio};
use std::sync::Mutex;
use std::thread;
use std::time::Duration;
use std::time::{SystemTime, UNIX_EPOCH};
use tauri::{Manager, State};
use tauri_plugin_updater::UpdaterExt;

const DEFAULT_BACKEND_URL: &str = "http://127.0.0.1:8080";
const DEFAULT_WEBSITE_URL: &str = "https://veltryx.karyuu.be";
const DEFAULT_SESSION_REGISTRY_URL: &str = "https://api-veltryx-1.karyuu.be";
const SESSION_TTL_SECS: u64 = 12 * 60 * 60;

struct AppState {
    backend: Client,
    website: Client,
    backend_url: String,
    website_url: String,
    session_registry_url: String,
    profile: Mutex<Option<DesktopIdentity>>,
    backend_process: Mutex<Option<Child>>,
    tms_primary_can_id: Option<i32>,
    tms_test_can_id: Option<i32>,
    tms_can_channel: String,
    session_key: Mutex<String>,
    client_label: String,
    cache_paths: CachePaths,
}

#[derive(Clone)]
struct CachePaths {
    root: PathBuf,
    session_file: PathBuf,
    key_file: PathBuf,
}

#[derive(Deserialize)]
struct DesktopLoginPayload {
    login: String,
    password: String,
}

#[derive(Clone, Default, Serialize, Deserialize)]
struct DesktopIdentity {
    id: String,
    username: String,
    email: String,
    name: String,
    role: String,
    #[serde(rename = "tenantId", default)]
    tenant_id: Option<String>,
    permissions: Vec<String>,
}

#[derive(Serialize, Deserialize)]
struct PersistedSession {
    version: u8,
    session_id: String,
    user_id: String,
    event_id: String,
    role: String,
    issued_at: u64,
    expires_at: u64,
    permissions_snapshot: Vec<String>,
    server_signature: String,
    identity: DesktopIdentity,
    device_id: String,
}

#[derive(Serialize, Deserialize)]
struct EncryptedSessionFile {
    version: u8,
    nonce: String,
    ciphertext: String,
}

#[derive(Serialize, Deserialize)]
struct ClientSessionUser {
    id: String,
    username: String,
    email: String,
    name: String,
    #[serde(rename = "tenantId", skip_serializing_if = "Option::is_none")]
    tenant_id: Option<String>,
    role: String,
    permissions: Vec<String>,
}

#[derive(Serialize, Deserialize)]
struct ClientSessionResponse {
    valid: bool,
    #[serde(default)]
    session_id: String,
    #[serde(default)]
    issued_at: Option<String>,
    #[serde(default)]
    expires_at: Option<String>,
    #[serde(default)]
    server_signature: String,
    #[serde(default)]
    reason: String,
    #[serde(default)]
    user: Option<ClientSessionUser>,
}

#[derive(Serialize)]
struct ValidateRequest<'a> {
    session_id: &'a str,
    device_id: &'a str,
    last_known_user: &'a str,
}

#[derive(Serialize)]
struct RegisterRequest<'a> {
    session_id: &'a str,
    device_id: &'a str,
    user: ClientSessionUser,
    expires_at: String,
}

fn build_http_client(timeout_secs: u64, cookie_store: bool) -> Result<Client, String> {
    Client::builder()
        .timeout(Duration::from_secs(timeout_secs))
        .cookie_store(cookie_store)
        .build()
        .map_err(|e| format!("http client failed: {e}"))
}

async fn backend_get(
    client: &Client,
    base_url: &str,
    identity: Option<DesktopIdentity>,
    path: &str,
) -> Result<Value, String> {
    let mut request = client.get(format!("{base_url}{path}"));
    if let Some(identity) = identity {
        request = request
            .header("X-Veltryx-Role", identity.role)
            .header("X-Veltryx-Permissions", identity.permissions.join(","));
    }
    let response = request
        .send()
        .await
        .map_err(|e| format!("backend request failed: {e}"))?;
    let status = response.status();
    let body = read_json_or_text(response).await?;

    if !status.is_success() {
        return Err(body
            .get("error")
            .and_then(Value::as_str)
            .or_else(|| body.get("raw").and_then(Value::as_str))
            .unwrap_or("backend request failed")
            .to_string());
    }

    Ok(body)
}

async fn backend_post_value(
    client: &Client,
    base_url: &str,
    identity: Option<DesktopIdentity>,
    path: &str,
    payload: Value,
) -> Result<Value, String> {
    let mut request = client.post(format!("{base_url}{path}")).json(&payload);
    if let Some(identity) = identity {
        request = request
            .header("X-Veltryx-Role", identity.role)
            .header("X-Veltryx-Permissions", identity.permissions.join(","));
    }
    let response = request
        .send()
        .await
        .map_err(|e| format!("backend request failed: {e}"))?;
    let status = response.status();
    let body = read_json_or_text(response).await?;

    if !status.is_success() {
        return Err(body
            .get("reason")
            .and_then(Value::as_str)
            .or_else(|| body.get("error").and_then(Value::as_str))
            .or_else(|| body.get("raw").and_then(Value::as_str))
            .unwrap_or("backend request failed")
            .to_string());
    }

    Ok(body)
}

async fn backend_post(
    client: &Client,
    base_url: &str,
    identity: Option<DesktopIdentity>,
    path: &str,
) -> Result<Value, String> {
    backend_post_value(client, base_url, identity, path, json!({})).await
}

async fn website_get(client: &Client, base_url: &str, path: &str) -> Result<Value, String> {
    let response = client
        .get(format!("{base_url}{path}"))
        .send()
        .await
        .map_err(|e| format!("website request failed: {e}"))?;
    let status = response.status();
    let body: Value = response
        .json()
        .await
        .map_err(|e| format!("invalid website json: {e}"))?;

    if !status.is_success() {
        return Err(body
            .get("error")
            .and_then(Value::as_str)
            .unwrap_or("website request failed")
            .to_string());
    }

    Ok(body)
}

async fn read_json_or_text(response: reqwest::Response) -> Result<Value, String> {
    let raw = response
        .text()
        .await
        .map_err(|e| format!("failed to read backend response: {e}"))?;

    match serde_json::from_str::<Value>(&raw) {
        Ok(value) => Ok(value),
        Err(_) => Ok(json!({ "raw": raw })),
    }
}

async fn website_post(client: &Client, base_url: &str, path: &str, payload: Value) -> Result<Value, String> {
    let response = client
        .post(format!("{base_url}{path}"))
        .json(&payload)
        .send()
        .await
        .map_err(|e| format!("website request failed: {e}"))?;
    let status = response.status();
    let body: Value = response
        .json()
        .await
        .map_err(|e| format!("invalid website json: {e}"))?;

    if !status.is_success() {
        return Err(body
            .get("error")
            .and_then(Value::as_str)
            .unwrap_or("website request failed")
            .to_string());
    }

    Ok(body)
}

fn identity_from_profile(payload: &Value) -> Option<DesktopIdentity> {
    let user = payload.get("user")?;
    let role = user.get("role")?.as_str()?.to_string();
    let permissions = user
        .get("permissions")?
        .as_array()?
        .iter()
        .filter_map(|entry| entry.as_str().map(ToString::to_string))
        .collect::<Vec<_>>();

    let username = user
        .get("username")
        .and_then(Value::as_str)
        .unwrap_or_default()
        .to_string();
    let email = user
        .get("email")
        .and_then(Value::as_str)
        .unwrap_or_default()
        .to_string();
    let name = user
        .get("name")
        .and_then(Value::as_str)
        .unwrap_or_else(|| if username.is_empty() { "Desktop user" } else { username.as_str() })
        .to_string();
    let id = user
        .get("id")
        .and_then(Value::as_str)
        .map(ToString::to_string)
        .or_else(|| (!username.is_empty()).then(|| username.clone()))
        .or_else(|| (!email.is_empty()).then(|| email.clone()))?;

    Some(DesktopIdentity {
        id,
        username,
        email,
        name,
        role,
        tenant_id: user
            .get("tenantId")
            .and_then(Value::as_str)
            .map(ToString::to_string),
        permissions,
    })
}

fn desktop_permissions_from_role(role: &str) -> Vec<String> {
    match role {
        "super_admin" => vec![
            "desktop:admin".to_string(),
            "desktop:control".to_string(),
            "desktop:view".to_string(),
        ],
        "admin" => vec!["desktop:control".to_string(), "desktop:view".to_string()],
        _ => vec!["desktop:view".to_string()],
    }
}

fn normalize_website_profile(payload: Value) -> Value {
    let authenticated = payload
        .get("authenticated")
        .and_then(Value::as_bool)
        .unwrap_or(false);

    if !authenticated {
        return json!({
            "authenticated": false,
            "user": null,
        });
    }

    let Some(user) = payload.get("user") else {
        return json!({
            "authenticated": false,
            "user": null,
        });
    };

    let role = user
        .get("role")
        .and_then(Value::as_str)
        .unwrap_or("user")
        .to_string();

    let permissions = user
        .get("permissions")
        .and_then(Value::as_array)
        .map(|entries| {
            entries
                .iter()
                .filter_map(|entry| entry.as_str().map(ToString::to_string))
                .collect::<Vec<_>>()
        })
        .filter(|entries| !entries.is_empty())
        .unwrap_or_else(|| desktop_permissions_from_role(&role));

    json!({
        "authenticated": true,
        "user": {
            "id": user.get("id").and_then(Value::as_str).unwrap_or_default(),
            "username": user.get("username").and_then(Value::as_str).unwrap_or_default(),
            "email": user.get("email").and_then(Value::as_str).unwrap_or_default(),
            "name": user.get("name").and_then(Value::as_str).unwrap_or_default(),
            "role": role,
            "tenantId": user.get("tenantId").cloned().unwrap_or(Value::Null),
            "permissions": permissions,
        }
    })
}

async fn fetch_desktop_profile(client: &Client, base_url: &str) -> Result<Value, String> {
    match website_get(client, base_url, "/api/auth/desktop/profile").await {
        Ok(profile) => Ok(normalize_website_profile(profile)),
        Err(primary_error) => {
            let session = website_get(client, base_url, "/api/auth/session")
                .await
                .map(normalize_website_profile);

            match session {
                Ok(profile) => Ok(profile),
                Err(_) => Err(primary_error),
            }
        }
    }
}

fn profile_value(identity: &DesktopIdentity) -> Value {
    json!({
        "authenticated": true,
        "user": {
            "id": identity.id,
            "username": identity.username,
            "email": identity.email,
            "name": identity.name,
            "role": identity.role,
            "tenantId": identity.tenant_id,
            "permissions": identity.permissions,
        }
    })
}

fn get_identity(state: &State<'_, AppState>) -> Option<DesktopIdentity> {
    state.profile.lock().ok()?.clone()
}

fn build_session_key() -> String {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_millis())
        .unwrap_or_default();
    format!("desktop-{now}")
}

fn now_epoch() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_secs())
        .unwrap_or_default()
}

fn env_i32(name: &str) -> Option<i32> {
    std::env::var(name).ok()?.trim().parse::<i32>().ok()
}

fn default_backend_socket() -> SocketAddr {
    SocketAddr::from(([127, 0, 0, 1], 8080))
}

fn is_backend_available() -> bool {
    TcpStream::connect_timeout(&default_backend_socket(), Duration::from_millis(250)).is_ok()
}

fn wait_for_backend_ready(timeout: Duration) -> bool {
    let started_at = SystemTime::now();
    while started_at
        .elapsed()
        .map(|elapsed| elapsed < timeout)
        .unwrap_or(false)
    {
        if is_backend_available() {
            return true;
        }
        thread::sleep(Duration::from_millis(250));
    }

    is_backend_available()
}

fn resolve_embedded_backend_path<R: tauri::Runtime>(app: &tauri::AppHandle<R>) -> Option<PathBuf> {
    let resource_dir = app.path().resource_dir().ok()?;
    [
        resource_dir.join("api.exe"),
        resource_dir.join("api"),
        resource_dir.join("bin").join("api.exe"),
        resource_dir.join("bin").join("api"),
    ]
    .into_iter()
    .find(|candidate| candidate.exists())
}

fn stop_embedded_backend(state: &State<'_, AppState>) {
    if let Ok(mut process) = state.backend_process.lock() {
        if let Some(child) = process.as_mut() {
            let _ = child.kill();
            let _ = child.wait();
        }
        *process = None;
    }
}

fn ensure_embedded_backend<R: tauri::Runtime>(
    app: &tauri::AppHandle<R>,
    state: &State<'_, AppState>,
) -> Result<(), String> {
    if cfg!(debug_assertions) || state.backend_url != DEFAULT_BACKEND_URL || is_backend_available() {
        return Ok(());
    }

    let backend_path = resolve_embedded_backend_path(app)
        .ok_or_else(|| "embedded backend binary not found in bundle resources".to_string())?;

    let mut process = state
        .backend_process
        .lock()
        .map_err(|_| "backend process lock failed".to_string())?;

    if process.is_some() {
        return Ok(());
    }

    let child = Command::new(&backend_path)
        .current_dir(
            backend_path
                .parent()
                .ok_or_else(|| "embedded backend directory missing".to_string())?,
        )
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .map_err(|e| format!("failed to launch embedded backend: {e}"))?;

    *process = Some(child);
    drop(process);

    if wait_for_backend_ready(Duration::from_secs(8)) {
        Ok(())
    } else {
        stop_embedded_backend(state);
        Err("embedded backend did not become ready on 127.0.0.1:8080".to_string())
    }
}

fn cache_paths() -> Result<CachePaths, String> {
    let appdata = std::env::var("APPDATA").map_err(|_| "APPDATA is not defined".to_string())?;
    let root = PathBuf::from(appdata).join("veltryx");
    Ok(CachePaths {
        session_file: root.join("session.dat"),
        key_file: root.join("device.key"),
        root,
    })
}

fn ensure_cache_root(paths: &CachePaths) -> Result<(), String> {
    fs::create_dir_all(&paths.root).map_err(|e| format!("failed to create cache directory: {e}"))
}

fn load_or_create_key(paths: &CachePaths) -> Result<Vec<u8>, String> {
    ensure_cache_root(paths)?;
    if paths.key_file.exists() {
        let raw = fs::read_to_string(&paths.key_file)
            .map_err(|e| format!("failed to read device key: {e}"))?;
        return BASE64
            .decode(raw.trim())
            .map_err(|e| format!("failed to decode device key: {e}"));
    }

    let mut key = vec![0u8; 32];
    rand::thread_rng().fill_bytes(&mut key);
    let encoded = BASE64.encode(&key);
    fs::write(&paths.key_file, encoded).map_err(|e| format!("failed to write device key: {e}"))?;
    Ok(key)
}

fn device_id_from_key(key: &[u8]) -> String {
    let digest = Sha256::digest(key);
    let mut out = String::new();
    for byte in digest[..16].iter() {
        out.push_str(&format!("{byte:02x}"));
    }
    out
}

fn encrypt_session(paths: &CachePaths, session: &PersistedSession) -> Result<(), String> {
    let key = load_or_create_key(paths)?;
    let cipher = Aes256Gcm::new_from_slice(&key).map_err(|e| format!("cipher init failed: {e}"))?;
    let mut nonce_bytes = [0u8; 12];
    rand::thread_rng().fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let plaintext =
        serde_json::to_vec(session).map_err(|e| format!("session encode failed: {e}"))?;
    let ciphertext = cipher
        .encrypt(nonce, plaintext.as_ref())
        .map_err(|e| format!("session encryption failed: {e}"))?;
    let file = EncryptedSessionFile {
        version: 1,
        nonce: BASE64.encode(nonce_bytes),
        ciphertext: BASE64.encode(ciphertext),
    };
    let raw = serde_json::to_vec(&file).map_err(|e| format!("session file encode failed: {e}"))?;
    fs::write(&paths.session_file, raw)
        .map_err(|e| format!("failed to write session cache: {e}"))?;
    Ok(())
}

fn load_cached_session(paths: &CachePaths) -> Result<Option<PersistedSession>, String> {
    if !paths.session_file.exists() {
        return Ok(None);
    }
    let key = load_or_create_key(paths)?;
    let raw =
        fs::read(&paths.session_file).map_err(|e| format!("failed to read session cache: {e}"))?;
    let file: EncryptedSessionFile =
        serde_json::from_slice(&raw).map_err(|e| format!("invalid session cache format: {e}"))?;
    let nonce_bytes = BASE64
        .decode(file.nonce)
        .map_err(|e| format!("invalid session nonce: {e}"))?;
    let ciphertext = BASE64
        .decode(file.ciphertext)
        .map_err(|e| format!("invalid session ciphertext: {e}"))?;
    let cipher = Aes256Gcm::new_from_slice(&key).map_err(|e| format!("cipher init failed: {e}"))?;
    let plaintext = cipher
        .decrypt(Nonce::from_slice(&nonce_bytes), ciphertext.as_ref())
        .map_err(|_| "session cache integrity check failed".to_string())?;
    let session: PersistedSession =
        serde_json::from_slice(&plaintext).map_err(|e| format!("invalid cached session: {e}"))?;
    Ok(Some(session))
}

fn clear_cached_session(paths: &CachePaths) -> Result<(), String> {
    if paths.session_file.exists() {
        fs::remove_file(&paths.session_file)
            .map_err(|e| format!("failed to clear session cache: {e}"))?;
    }
    Ok(())
}

fn parse_server_time(value: Option<String>) -> u64 {
    value
        .and_then(|v| chrono_like_parse(&v))
        .unwrap_or_else(now_epoch)
}

fn chrono_like_parse(value: &str) -> Option<u64> {
    let parsed = time_like::parse_rfc3339(value)?;
    Some(parsed)
}

mod time_like {
    use std::time::{SystemTime, UNIX_EPOCH};

    pub fn parse_rfc3339(input: &str) -> Option<u64> {
        let parsed =
            time::OffsetDateTime::parse(input, &time::format_description::well_known::Rfc3339)
                .ok()?;
        let unix = parsed.unix_timestamp();
        if unix < 0 {
            return None;
        }
        Some(unix as u64)
    }

    pub fn to_rfc3339(epoch_secs: u64) -> String {
        let dt = UNIX_EPOCH + std::time::Duration::from_secs(epoch_secs);
        let st: SystemTime = dt;
        let odt = time::OffsetDateTime::from(st);
        odt.format(&time::format_description::well_known::Rfc3339)
            .unwrap_or_else(|_| "1970-01-01T00:00:00Z".to_string())
    }
}

fn build_cached_session(
    session_id: String,
    event_id: String,
    identity: DesktopIdentity,
    validation: &ClientSessionResponse,
    device_id: String,
) -> PersistedSession {
    let issued_at = parse_server_time(validation.issued_at.clone());
    let expires_at = parse_server_time(validation.expires_at.clone());
    PersistedSession {
        version: 1,
        session_id,
        user_id: identity.id.clone(),
        event_id,
        role: identity.role.clone(),
        issued_at,
        expires_at,
        permissions_snapshot: identity.permissions.clone(),
        server_signature: validation.server_signature.clone(),
        identity,
        device_id,
    }
}

async fn register_remote_session(
    state: &State<'_, AppState>,
    session_id: &str,
    identity: &DesktopIdentity,
    device_id: &str,
) -> Result<ClientSessionResponse, String> {
    let expires_at = now_epoch() + SESSION_TTL_SECS;
    let payload = RegisterRequest {
        session_id,
        device_id,
        user: ClientSessionUser {
            id: identity.id.clone(),
            username: identity.username.clone(),
            email: identity.email.clone(),
            name: identity.name.clone(),
            tenant_id: identity.tenant_id.clone(),
            role: identity.role.clone(),
            permissions: identity.permissions.clone(),
        },
        expires_at: time_like::to_rfc3339(expires_at),
    };
    let value = backend_post_value(
        &state.backend,
        &state.session_registry_url,
        None,
        "/api/client-session/register",
        serde_json::to_value(payload).map_err(|e| format!("register payload failed: {e}"))?,
    )
    .await?;
    serde_json::from_value(value).map_err(|e| format!("invalid register response: {e}"))
}

async fn validate_remote_session(
    state: &State<'_, AppState>,
    cached: &PersistedSession,
    device_id: &str,
) -> Result<ClientSessionResponse, String> {
    let payload = ValidateRequest {
        session_id: &cached.session_id,
        device_id,
        last_known_user: &cached.user_id,
    };
    let value = backend_post_value(
        &state.backend,
        &state.session_registry_url,
        None,
        "/api/client-session/validate",
        serde_json::to_value(payload).map_err(|e| format!("validate payload failed: {e}"))?,
    )
    .await?;
    serde_json::from_value(value).map_err(|e| format!("invalid validate response: {e}"))
}

async fn heartbeat_presence(state: &State<'_, AppState>, leave: bool) -> Result<(), String> {
    website_post(
        &state.website,
        &state.website_url,
        "/api/auth/desktop/presence",
        json!({
            "sessionKey": state.session_key.lock().map_err(|_| "session key lock failed".to_string())?.clone(),
            "clientLabel": state.client_label,
            "leave": leave,
        }),
    )
    .await
    .map(|_| ())
}

#[tauri::command]
async fn get_x2_telemetry(state: State<'_, AppState>) -> Result<Value, String> {
    backend_get(
        &state.backend,
        &state.backend_url,
        get_identity(&state),
        "/api/x2/telemetry",
    )
    .await
}

#[tauri::command]
async fn get_live_timing_snapshot(state: State<'_, AppState>) -> Result<Value, String> {
    backend_get(
        &state.backend,
        &state.backend_url,
        get_identity(&state),
        "/api/livetiming/snapshot",
    )
    .await
}

#[tauri::command]
async fn connect_x2(state: State<'_, AppState>) -> Result<Value, String> {
    backend_post(
        &state.backend,
        &state.backend_url,
        get_identity(&state),
        "/api/x2/connect",
    )
    .await
}

#[tauri::command]
async fn disconnect_x2(state: State<'_, AppState>) -> Result<Value, String> {
    backend_post(
        &state.backend,
        &state.backend_url,
        get_identity(&state),
        "/api/x2/disconnect",
    )
    .await
}

#[tauri::command]
async fn set_global_flag(state: State<'_, AppState>, flag: i32) -> Result<Value, String> {
    backend_post_value(
        &state.backend,
        &state.backend_url,
        get_identity(&state),
        "/api/x2/flags/global",
        json!({ "flag": flag }),
    )
    .await
}

#[tauri::command]
async fn set_sector_flag(state: State<'_, AppState>, sector_id: i32, flag: i32) -> Result<Value, String> {
    backend_post_value(
        &state.backend,
        &state.backend_url,
        get_identity(&state),
        "/api/x2/flags/sector",
        json!({ "sectorId": sector_id, "flag": flag }),
    )
    .await
}

#[tauri::command]
async fn get_tms_screen_config(state: State<'_, AppState>) -> Result<Value, String> {
    Ok(json!({
        "primaryCanId": state.tms_primary_can_id,
        "testCanId": state.tms_test_can_id,
        "channel": state.tms_can_channel,
    }))
}

#[tauri::command]
async fn send_tms_can(
    state: State<'_, AppState>,
    data: Vec<i32>,
    primary_can_id: Option<i32>,
    test_can_id: Option<i32>,
    can_id: Option<i32>,
) -> Result<Value, String> {
    if data.is_empty() {
        return Err("missing can payload".to_string());
    }

    let selected_can_id = can_id.unwrap_or(514);
    if data.iter().any(|value| *value < 0 || *value > 255) {
        return Err("invalid can payload byte".to_string());
    }
    let payload = data;

    let targets = [primary_can_id.or(state.tms_primary_can_id), test_can_id.or(state.tms_test_can_id)];
    let target_ids = targets
        .into_iter()
        .flatten()
        .filter(|id| *id > 0)
        .collect::<Vec<_>>();
    let mut unique_target_ids = Vec::with_capacity(target_ids.len());
    for target_id in target_ids {
        if !unique_target_ids.contains(&target_id) {
            unique_target_ids.push(target_id);
        }
    }

    if unique_target_ids.is_empty() {
        return Err("no TMS CAN targets configured".to_string());
    }

    let mut sent = Vec::with_capacity(unique_target_ids.len());
    for can_id in unique_target_ids {
        backend_post_value(
            &state.backend,
            &state.backend_url,
            get_identity(&state),
            "/api/x2/send-can",
            json!({
                "id": can_id,
                "data": payload,
                "canId": selected_can_id,
            }),
        )
        .await?;
        sent.push(can_id);
    }

    Ok(json!({
        "ok": true,
        "canId": selected_can_id,
        "data": payload,
        "targets": sent,
    }))
}

#[tauri::command]
async fn get_x2_status(state: State<'_, AppState>) -> Result<Value, String> {
    match backend_get(
        &state.backend,
        &state.backend_url,
        get_identity(&state),
        "/api/x2/status",
    )
    .await
    {
        Ok(value) => Ok(value),
        Err(err) => Ok(json!({
            "connected": false,
            "state": format!("backend unavailable: {err}")
        })),
    }
}

#[tauri::command]
async fn bootstrap_desktop_session(state: State<'_, AppState>) -> Result<Value, String> {
    let key = load_or_create_key(&state.cache_paths)?;
    let device_id = device_id_from_key(&key);
    let cached = match load_cached_session(&state.cache_paths)? {
        Some(cached) => cached,
        None => {
            return Ok(json!({
                "authenticated": false,
                "user": null,
            }))
        }
    };

    if cached.device_id != device_id || cached.expires_at <= now_epoch() {
        let _ = clear_cached_session(&state.cache_paths);
        return Ok(json!({
            "authenticated": false,
            "user": null,
        }));
    }

    match validate_remote_session(&state, &cached, &device_id).await {
        Ok(response) if response.valid => {
            let identity = match &response.user {
                Some(user) => DesktopIdentity {
                    id: user.id.clone(),
                    username: user.username.clone(),
                    email: user.email.clone(),
                    name: user.name.clone(),
                    role: user.role.clone(),
                    tenant_id: user.tenant_id.clone(),
                    permissions: user.permissions.clone(),
                },
                None => {
                    let _ = clear_cached_session(&state.cache_paths);
                    return Ok(json!({"authenticated": false, "user": null}));
                }
            };

            let refreshed = build_cached_session(
                cached.session_id.clone(),
                cached.event_id,
                identity.clone(),
                &response,
                device_id,
            );
            encrypt_session(&state.cache_paths, &refreshed)?;
            if let Ok(mut current) = state.profile.lock() {
                *current = Some(identity.clone());
            }
            if let Ok(mut key_guard) = state.session_key.lock() {
                *key_guard = refreshed.session_id.clone();
            }
            Ok(profile_value(&identity))
        }
        _ => {
            let _ = clear_cached_session(&state.cache_paths);
            if let Ok(mut current) = state.profile.lock() {
                *current = None;
            }
            Ok(json!({
                "authenticated": false,
                "user": null,
            }))
        }
    }
}

#[tauri::command]
async fn desktop_login(
    state: State<'_, AppState>,
    payload: DesktopLoginPayload,
) -> Result<Value, String> {
    website_post(
        &state.website,
        &state.website_url,
        "/api/auth/login",
        json!({
            "login": payload.login,
            "password": payload.password,
        }),
    )
    .await?;

    let profile = fetch_desktop_profile(&state.website, &state.website_url).await?;
    if profile
        .get("authenticated")
        .and_then(Value::as_bool)
        != Some(true)
    {
        return Err("desktop profile not authenticated after login".to_string());
    }

    let identity = identity_from_profile(&profile).ok_or_else(|| {
        format!(
            "invalid desktop profile: {}",
            serde_json::to_string(&profile).unwrap_or_else(|_| "unserializable_profile".to_string())
        )
    })?;
    let key = load_or_create_key(&state.cache_paths)?;
    let device_id = device_id_from_key(&key);
    let event_id = "desktop".to_string();
    let session_id = build_session_key();
    let validation = register_remote_session(&state, &session_id, &identity, &device_id).await?;
    let cached = build_cached_session(
        session_id.clone(),
        event_id,
        identity.clone(),
        &validation,
        device_id,
    );
    encrypt_session(&state.cache_paths, &cached)?;
    if let Ok(mut current) = state.profile.lock() {
        *current = Some(identity);
    }
    if let Ok(mut key_guard) = state.session_key.lock() {
        *key_guard = session_id;
    }
    let _ = heartbeat_presence(&state, false).await;
    Ok(profile)
}

#[tauri::command]
async fn desktop_logout(state: State<'_, AppState>) -> Result<Value, String> {
    let session_id = state
        .session_key
        .lock()
        .map_err(|_| "session key lock failed".to_string())?
        .clone();
    let _ = heartbeat_presence(&state, true).await;
    let _ = backend_post_value(
        &state.backend,
        &state.session_registry_url,
        None,
        "/api/client-session/logout",
        json!({ "session_id": session_id }),
    )
    .await;
    let _ = website_post(&state.website, &state.website_url, "/api/auth/logout", json!({})).await;
    clear_cached_session(&state.cache_paths)?;
    if let Ok(mut current) = state.profile.lock() {
        *current = None;
    }
    if let Ok(mut key_guard) = state.session_key.lock() {
        *key_guard = build_session_key();
    }
    Ok(json!({ "ok": true }))
}

#[tauri::command]
async fn get_desktop_profile(state: State<'_, AppState>) -> Result<Value, String> {
    if let Some(identity) = get_identity(&state) {
        return Ok(profile_value(&identity));
    }

    match fetch_desktop_profile(&state.website, &state.website_url).await {
        Ok(value) => {
            if let Some(identity) = identity_from_profile(&value) {
                if let Ok(mut current) = state.profile.lock() {
                    *current = Some(identity);
                }
                let _ = heartbeat_presence(&state, false).await;
            }
            Ok(value)
        }
        Err(_) => Ok(json!({
            "authenticated": false,
            "user": null,
        })),
    }
}

#[tauri::command]
async fn get_admin_desktop_presence(state: State<'_, AppState>) -> Result<Value, String> {
    backend_get(
        &state.backend,
        &state.session_registry_url,
        get_identity(&state),
        "/api/client-session/list",
    )
    .await
}

#[tauri::command]
async fn disconnect_desktop_session(
    state: State<'_, AppState>,
    session_id: String,
) -> Result<Value, String> {
    backend_post_value(
        &state.backend,
        &state.session_registry_url,
        get_identity(&state),
        "/api/client-session/logout",
        json!({ "session_id": session_id }),
    )
    .await?;

    let current_session_id = state
        .session_key
        .lock()
        .map_err(|_| "session key lock failed".to_string())?
        .clone();

    if session_id == current_session_id {
        let _ = website_post(&state.website, &state.website_url, "/api/auth/logout", json!({})).await;
        let _ = clear_cached_session(&state.cache_paths);
        if let Ok(mut current) = state.profile.lock() {
            *current = None;
        }
        if let Ok(mut key_guard) = state.session_key.lock() {
            *key_guard = build_session_key();
        }
        return Ok(json!({ "ok": true, "self": true }));
    }

    Ok(json!({ "ok": true }))
}

#[tauri::command]
async fn updater_check(app: tauri::AppHandle) -> Result<Value, String> {
    let updater = app
        .updater()
        .map_err(|e| format!("updater init failed: {e}"))?;
    let update = updater
        .check()
        .await
        .map_err(|e| format!("updater check failed: {e}"))?;

    if let Some(update) = update {
        let date = update.date.map(|d| d.to_string());
        return Ok(json!({
            "available": true,
            "version": format!("{}", update.version),
            "body": update.body,
            "date": date,
        }));
    }

    Ok(json!({
        "available": false,
    }))
}

#[tauri::command]
async fn updater_install(app: tauri::AppHandle) -> Result<Value, String> {
    let updater = app
        .updater()
        .map_err(|e| format!("updater init failed: {e}"))?;
    let update = updater
        .check()
        .await
        .map_err(|e| format!("updater check failed: {e}"))?;

    let Some(update) = update else {
        return Ok(json!({
            "updated": false,
            "reason": "no_update",
        }));
    };

    let version = format!("{}", update.version);
    update
        .download_and_install(
            |_chunk_length, _content_length| {},
            || {},
        )
        .await
        .map_err(|e| format!("updater install failed: {e}"))?;

    Ok(json!({
        "updated": true,
        "version": version,
        "restartRequired": true,
    }))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let backend = build_http_client(5, false).expect("failed to create backend client");
    let website = build_http_client(5, true).expect("failed to create website client");
    let backend_url =
        std::env::var("VELTRYX_BACKEND_URL").unwrap_or_else(|_| DEFAULT_BACKEND_URL.to_string());
    let website_url =
        std::env::var("VELTRYX_WEBSITE_URL").unwrap_or_else(|_| DEFAULT_WEBSITE_URL.to_string());
    let session_registry_url = std::env::var("VELTRYX_SESSION_REGISTRY_URL")
        .unwrap_or_else(|_| DEFAULT_SESSION_REGISTRY_URL.to_string());
    let tms_primary_can_id = env_i32("VELTRYX_TMS_PRIMARY_CAN_ID");
    let tms_test_can_id = env_i32("VELTRYX_TMS_TEST_CAN_ID");
    let tms_can_channel =
        std::env::var("VELTRYX_TMS_CAN_CHANNEL").unwrap_or_else(|_| "tms".to_string());
    let paths = cache_paths().expect("failed to resolve appdata paths");
    let _ = ensure_cache_root(&paths);
    let session_key = build_session_key();
    let client_label = std::env::var("COMPUTERNAME").unwrap_or_else(|_| "desktop".to_string());

    let app = tauri::Builder::default()
        .manage(AppState {
            backend,
            website,
            backend_url,
            website_url,
            session_registry_url,
            profile: Mutex::new(None),
            backend_process: Mutex::new(None),
            tms_primary_can_id,
            tms_test_can_id,
            tms_can_channel,
            session_key: Mutex::new(session_key),
            client_label,
            cache_paths: paths,
        })
        .setup(|app| {
            let state = app.state::<AppState>();
            ensure_embedded_backend(app.handle(), &state)
                .map_err(|e| -> Box<dyn std::error::Error> { e.into() })?;
            Ok(())
        })
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            bootstrap_desktop_session,
            get_x2_telemetry,
            get_live_timing_snapshot,
            connect_x2,
            disconnect_x2,
            get_tms_screen_config,
            send_tms_can,
            set_global_flag,
            set_sector_flag,
            get_x2_status,
            desktop_login,
            desktop_logout,
            get_desktop_profile,
            get_admin_desktop_presence,
            disconnect_desktop_session,
            updater_check,
            updater_install
        ])
        .build(tauri::generate_context!())
        .expect("error while building tauri application");

    app.run(|app_handle, event| match event {
        tauri::RunEvent::Exit | tauri::RunEvent::ExitRequested { .. } => {
            let state = app_handle.state::<AppState>();
            stop_embedded_backend(&state);
        }
        _ => {}
    });
}
