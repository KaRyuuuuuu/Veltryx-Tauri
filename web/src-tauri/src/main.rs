// Empêche l'ouverture d'une fenêtre de console supplémentaire sur Windows en mode release, NE PAS SUPPRIMER !!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    web_lib::run()
}
