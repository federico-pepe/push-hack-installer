import {WML} from "@wailsio/runtime";
import {ConnectService} from "../bindings/github.com/federico-pepe/push-hack-installer/cmd/installer-ui";

WML.Enable();

document.getElementById('version')!.innerText = "v3.0.0-beta.9";

// ----- Screen switching -------------------------------------------------
// Two screens (Welcome, Connect) rendered inline in index.html; only one is
// shown at a time via the .is-active class. No router needed at this size —
// see plans/2026-09-27-gui-installer.md for the rest of the planned flow
// (SSH key setup, hack selection, install progress, done).
function showScreen(id: string) {
    document.querySelectorAll<HTMLElement>('.screen').forEach((el) => {
        el.classList.toggle('is-active', el.id === id);
    });
}

document.getElementById('welcome-continue')!.addEventListener('click', () => {
    showScreen('screen-connect');
    checkConnection();
});

// ----- Connect screen -----------------------------------------------------
const hostInput = document.getElementById('host')! as HTMLInputElement;
const statusLine = document.getElementById('connect-status')! as HTMLParagraphElement;
const continueButton = document.getElementById('connect-continue')! as HTMLButtonElement;
const cantConnectToggle = document.getElementById('cant-connect-toggle')! as HTMLButtonElement;
const cantConnectHint = document.getElementById('cant-connect-hint')! as HTMLParagraphElement;

let checkToken = 0;

// Re-checks reachability against the current value of the host field. Each
// call invalidates any check still in flight (checkToken) so a slow lookup
// for a stale host can't overwrite a newer result — the user may have
// already fixed a typo and triggered a second check before the first one
// returns.
async function checkConnection() {
    const host = hostInput.value.trim();
    const token = ++checkToken;

    statusLine.textContent = 'Checking...';
    statusLine.className = 'status-line';
    continueButton.disabled = true;

    if (!host) {
        return;
    }

    try {
        const [reachable, reason] = await ConnectService.CheckHost(host);
        if (token !== checkToken) {
            return; // a newer check has since started
        }
        if (reachable) {
            statusLine.textContent = `Found Push at ${host}.`;
            statusLine.className = 'status-line is-ok';
            continueButton.disabled = false;
        } else {
            statusLine.textContent = reason;
            statusLine.className = 'status-line is-error';
        }
    } catch (err) {
        if (token !== checkToken) {
            return;
        }
        console.error(err);
        statusLine.textContent = "Couldn't check the connection. Try again.";
        statusLine.className = 'status-line is-error';
    }
}

let debounceTimer: ReturnType<typeof setTimeout>;
hostInput.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(checkConnection, 500);
});

cantConnectToggle.addEventListener('click', () => {
    cantConnectHint.hidden = !cantConnectHint.hidden;
});

continueButton.addEventListener('click', () => {
    // Next screen (SSH key setup) is not built yet.
    console.log('Connect continue clicked — next screen not implemented yet.');
});
