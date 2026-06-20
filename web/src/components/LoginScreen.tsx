import { motion } from "motion/react";

type LoginScreenProps = {
  shellTone: string;
  busy: boolean;
  authError: string;
  login: string;
  password: string;
  setLogin: (value: string) => void;
  setPassword: (value: string) => void;
  handleLogin: (event: React.FormEvent<HTMLFormElement>) => Promise<void>;
};

export function LoginScreen({
  shellTone,
  busy,
  authError,
  login,
  password,
  setLogin,
  setPassword,
  handleLogin,
}: LoginScreenProps) {
  return (
    <motion.div
      className={`login-shell ${shellTone}`}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.28, ease: "easeOut" }}
    >
      <motion.section
        className="login-card"
        initial={{ opacity: 0, y: 22, scale: 0.985 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 14, scale: 0.99 }}
        transition={{ duration: 0.32, ease: "easeOut" }}
      >
        <motion.div
          className="login-branding"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.28, delay: 0.03, ease: "easeOut" }}
        >
          <img className="login-logo-image" src="/branding/veltryx_icon.png" alt="Veltryx icon" />
        </motion.div>
        <motion.p
          className="eyebrow"
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.28, delay: 0.05, ease: "easeOut" }}
        >
          Veltryx Unified Desktop
        </motion.p>
        <motion.h1
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.08, ease: "easeOut" }}
        >
          Sign in to load your workspace
        </motion.h1>
        <form className="login-form" onSubmit={(event) => void handleLogin(event)}>
          <input
            value={login}
            onChange={(event) => setLogin(event.currentTarget.value)}
            placeholder="Email or username"
            autoComplete="username"
          />
          <input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.currentTarget.value)}
            placeholder="Password"
            autoComplete="current-password"
          />
          <button className="primary-button" type="submit" disabled={busy}>
            {busy ? "Connecting..." : "Login"}
          </button>
          {authError ? <p className="status-error">{authError}</p> : null}
        </form>
      </motion.section>
    </motion.div>
  );
}
