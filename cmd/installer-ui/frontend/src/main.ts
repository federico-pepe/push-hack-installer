import {WML} from "@wailsio/runtime";

// Wire up data-wml-openURL links, if any get added later.
WML.Enable();

// Show the Wails version this project was generated against.
document.getElementById('version')!.innerText = "v3.0.0-beta.9";

// Placeholder: the Welcome screen's "Continue" button has nowhere to go yet.
// The next screen (connect to Push over SSH) is not built. See
// plans/2026-09-27-gui-installer.md for the planned flow.
document.getElementById('continue')!.addEventListener('click', () => {
    console.log('Continue clicked — next screen not implemented yet.');
});
