import {Application, Clipboard, WML} from "@wailsio/runtime";
import {ConnectService, SSHKeyService} from "../bindings/github.com/federico-pepe/push-hack-installer/cmd/installer-ui";

WML.Enable();

// ----- Screen switching -------------------------------------------------
// Three screens (Welcome, Connect, SSH key setup) rendered inline in
// index.html; only one is shown at a time via the .is-active class. No
// router needed at this size — see plans/2026-09-27-gui-installer.md for
// the rest of the planned flow (hack selection, install progress, done).
function showScreen(id: string) {
    document.querySelectorAll<HTMLElement>('.screen').forEach((el) => {
        el.classList.toggle('is-active', el.id === id);
    });
}

// ----- Welcome screen -----------------------------------------------------
// Mirrors install.sh's disclaimer gate (type "Yes" to continue) — Continue
// stays disabled until the risk checkbox is ticked.
const acceptRiskCheckbox = document.getElementById('accept-risk')! as HTMLInputElement;
const welcomeContinueButton = document.getElementById('welcome-continue')! as HTMLButtonElement;
const welcomeCancelButton = document.getElementById('welcome-cancel')! as HTMLButtonElement;

acceptRiskCheckbox.addEventListener('change', () => {
    welcomeContinueButton.disabled = !acceptRiskCheckbox.checked;
});

welcomeContinueButton.addEventListener('click', () => {
    showScreen('screen-connect');
    checkConnection();
});

welcomeCancelButton.addEventListener('click', () => {
    Application.Quit();
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

let connectedHost = '';

continueButton.addEventListener('click', async () => {
    connectedHost = hostInput.value.trim();
    continueButton.disabled = true;
    statusLine.textContent = 'Checking for an existing key...';
    statusLine.className = 'status-line';

    let skipSSHKeySetup = false;
    try {
        skipSSHKeySetup = await SSHKeyService.AlreadyAuthorized(connectedHost);
    } catch (err) {
        console.error(err);
        // Not fatal — just means the SSH key screen runs as normal.
    }

    if (skipSSHKeySetup) {
        statusLine.textContent = 'Push already trusts this computer.';
        statusLine.className = 'status-line is-ok';
        continueButton.disabled = false;
        proceedPastSSHKeySetup();
    } else {
        continueButton.disabled = false;
        showScreen('screen-sshkey');
        startSSHKeySetup();
    }
});

// ----- SSH key screen -------------------------------------------------
const fingerprintEl = document.getElementById('fingerprint')! as HTMLElement;
const sshKeyStatus = document.getElementById('sshkey-status')! as HTMLParagraphElement;
const openSSHPageButton = document.getElementById('open-ssh-page')! as HTMLButtonElement;
const sshKeyContinueButton = document.getElementById('sshkey-continue')! as HTMLButtonElement;
const copyKeyButton = document.getElementById('copy-key')! as HTMLButtonElement;

let sshKeyPubLine = '';
let sshKeySetupStarted = false;

// Generates/loads the key once per app run — re-entering this screen (e.g.
// after "Try again") reuses the same key rather than regenerating it.
async function startSSHKeySetup() {
    if (sshKeySetupStarted) {
        return;
    }
    sshKeySetupStarted = true;

    try {
        const [pubKeyLine, fingerprint, err] = await SSHKeyService.EnsureKey();
        if (err) {
            fingerprintEl.textContent = 'Could not create a key.';
            sshKeyStatus.textContent = err;
            sshKeyStatus.className = 'status-line is-error';
            return;
        }
        sshKeyPubLine = pubKeyLine;
        fingerprintEl.textContent = fingerprint;
        openSSHPageButton.disabled = false;
        copyKeyButton.disabled = false;

        // Best-effort convenience copy so the common case ("generate, then
        // paste") needs no extra click. The explicit Copy key button below
        // is the reliable path if this silently does nothing.
        Clipboard.SetText(sshKeyPubLine).catch((err) => console.error(err));
    } catch (err) {
        console.error(err);
        fingerprintEl.textContent = 'Could not create a key.';
        sshKeyStatus.textContent = "Something went wrong generating the key.";
        sshKeyStatus.className = 'status-line is-error';
    }
}

copyKeyButton.addEventListener('click', async () => {
    try {
        await Clipboard.SetText(sshKeyPubLine);
        const original = copyKeyButton.textContent;
        copyKeyButton.textContent = 'Copied!';
        setTimeout(() => { copyKeyButton.textContent = original; }, 1500);
    } catch (err) {
        console.error(err);
        copyKeyButton.textContent = "Couldn't copy";
    }
});

openSSHPageButton.addEventListener('click', async () => {
    openSSHPageButton.disabled = true;
    sshKeyStatus.textContent = "Waiting for you to add the key on Push...";
    sshKeyStatus.className = 'status-line';

    try {
        await SSHKeyService.OpenSSHPage(connectedHost);
    } catch (err) {
        console.error(err);
        // Not fatal — the user can open the page manually, so polling still proceeds.
    }

    try {
        const [accepted, reason] = await SSHKeyService.WaitForKeyAccepted(connectedHost);
        if (accepted) {
            sshKeyStatus.textContent = 'Key accepted!';
            sshKeyStatus.className = 'status-line is-ok';
            openSSHPageButton.hidden = true;
            sshKeyContinueButton.hidden = false;
            sshKeyContinueButton.disabled = false;
        } else {
            sshKeyStatus.textContent = reason;
            sshKeyStatus.className = 'status-line is-error';
            openSSHPageButton.disabled = false;
        }
    } catch (err) {
        console.error(err);
        sshKeyStatus.textContent = "Couldn't check whether the key was accepted. Try again.";
        sshKeyStatus.className = 'status-line is-error';
        openSSHPageButton.disabled = false;
    }
});

// Reached either from the SSH key screen's own Continue button, or directly
// from the Connect screen when AlreadyAuthorized finds a previous run
// already got the key accepted for this host — see the connect-continue
// handler above.
function proceedPastSSHKeySetup() {
    // TEMPORARY: hack selection (the actual next screen) isn't built yet.
    // A visible placeholder here, not just a console.log, so reaching this
    // point during testing doesn't look like the app silently hung.
    console.log('Past SSH key setup — next screen not implemented yet.');
    alert("Push is set up and trusted. Hack selection isn't built yet — that's next.");
}

sshKeyContinueButton.addEventListener('click', proceedPastSSHKeySetup);
