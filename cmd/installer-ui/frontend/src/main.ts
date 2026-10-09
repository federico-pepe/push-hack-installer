import {Application, Clipboard, WML} from "@wailsio/runtime";
import {ConnectService, SSHKeyService, PushInstallService, LogService, UpdateService} from "../bindings/github.com/federico-pepe/push-hack-installer/cmd/installer-ui";

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
    showScreen('screen-install');
    checkInstallStatus();
}

sshKeyContinueButton.addEventListener('click', proceedPastSSHKeySetup);

// ----- Install / uninstall screen ------------------------------------------
// No hack-selection screen — always all three core hacks (push-manager,
// push-display, push-hack-catalog) together, per the user's call: this app
// installs push-hack's non-optional framework core, not a pick-list.
const installStatus = document.getElementById('install-status')! as HTMLParagraphElement;
const installStatusIcon = document.getElementById('install-status-icon')! as unknown as SVGElement & {hidden: boolean};
const installStatusText = document.getElementById('install-status-text')! as HTMLSpanElement;
const installLog = document.getElementById('install-log')! as HTMLUListElement;
const doInstallButton = document.getElementById('do-install')! as HTMLButtonElement;
const doUninstallButton = document.getElementById('do-uninstall')! as HTMLButtonElement;
const openAppActions = document.getElementById('open-app-actions')! as HTMLDivElement;
const openPushManagerButton = document.getElementById('open-push-manager')! as HTMLButtonElement;
const openPushCatalogButton = document.getElementById('open-push-catalog')! as HTMLButtonElement;

function setInstallStatus(text: string, kind: '' | 'ok' | 'error' = '') {
    installStatusText.textContent = text;
    installStatus.className = kind ? `status-line status-line-big is-${kind}` : 'status-line status-line-big';
    installStatusIcon.hidden = kind !== 'ok';
}

function renderInstallLog(lines: string[]) {
    installLog.innerHTML = '';
    for (const line of lines) {
        const li = document.createElement('li');
        li.textContent = line;
        installLog.appendChild(li);
    }
    installLog.hidden = lines.length === 0;
}

async function checkInstallStatus() {
    doInstallButton.hidden = false;
    doUninstallButton.hidden = false;
    doInstallButton.disabled = true;
    doUninstallButton.disabled = true;
    openAppActions.hidden = true;
    setInstallStatus('Checking whether push-hack is already on your Push...');

    try {
        const [installed, err] = await PushInstallService.IsInstalled(connectedHost);
        if (err) {
            setInstallStatus(err, 'error');
            doInstallButton.disabled = false; // let them try anyway
            doUninstallButton.disabled = false;
            return;
        }
        if (installed) {
            setInstallStatus('Push Hack is already installed on Push', 'ok');
            doInstallButton.disabled = true;
            doUninstallButton.disabled = false;
            openAppActions.hidden = false;
        } else {
            setInstallStatus('Push Hack is not installed on this Push yet.');
            doInstallButton.disabled = false;
            doUninstallButton.disabled = true;
        }
    } catch (err) {
        console.error(err);
        setInstallStatus('Something went wrong. Try again.', 'error');
    }
}

doInstallButton.addEventListener('click', async () => {
    doInstallButton.disabled = true;
    doUninstallButton.disabled = true;
    setInstallStatus('Installing... this restarts Push3, please wait.');
    renderInstallLog([]);

    try {
        const [summary, err] = await PushInstallService.Install(connectedHost);
        renderInstallLog(summary ?? []);
        if (err) {
            setInstallStatus(err, 'error');
            doInstallButton.disabled = false;
        } else {
            setInstallStatus('Push Hack is already installed on Push', 'ok');
            doUninstallButton.disabled = false;
            openAppActions.hidden = false;
        }
    } catch (err) {
        console.error(err);
        setInstallStatus('Something went wrong installing push-hack.', 'error');
        doInstallButton.disabled = false;
    }
});

doUninstallButton.addEventListener('click', async () => {
    doUninstallButton.disabled = true;
    doInstallButton.disabled = true;
    openAppActions.hidden = true;
    setInstallStatus('Uninstalling... this restarts Push3, please wait.');
    renderInstallLog([]);

    try {
        const [summary, err] = await PushInstallService.Uninstall(connectedHost);
        renderInstallLog(summary ?? []);
        if (err) {
            setInstallStatus(err, 'error');
            doUninstallButton.disabled = false;
        } else {
            setInstallStatus('Push Hack is not installed on this Push yet.');
            doInstallButton.disabled = false;
        }
    } catch (err) {
        console.error(err);
        setInstallStatus('Something went wrong removing push-hack.', 'error');
        doUninstallButton.disabled = false;
    }
});

openPushManagerButton.addEventListener('click', async () => {
    try {
        await PushInstallService.OpenPushManager(connectedHost);
    } catch (err) {
        console.error(err);
    }
});

openPushCatalogButton.addEventListener('click', async () => {
    try {
        await PushInstallService.OpenPushCatalog(connectedHost);
    } catch (err) {
        console.error(err);
    }
});

// ----- Footer: Export Debug Logs -------------------------------------------
// Shown outside the .screen elements, so it's reachable from every step of
// the flow, not just the install screen.
const exportLogsButton = document.getElementById('export-logs')! as HTMLButtonElement;

exportLogsButton.addEventListener('click', async () => {
    const original = exportLogsButton.textContent;
    exportLogsButton.disabled = true;
    exportLogsButton.textContent = 'Exporting...';

    try {
        const err = await LogService.Export();
        exportLogsButton.textContent = err ? "Couldn't export logs" : 'Logs exported';
    } catch (err) {
        console.error(err);
        exportLogsButton.textContent = "Couldn't export logs";
    } finally {
        setTimeout(() => {
            exportLogsButton.textContent = original;
            exportLogsButton.disabled = false;
        }, 2000);
    }
});

// ----- Update check ---------------------------------------------------------
// Runs once at startup. The banner stays hidden if the app is up to date,
// offline, or a dev build (Check returns "").
const updateBanner = document.getElementById('update-banner')! as HTMLDivElement;
const updateBannerText = document.getElementById('update-banner-text')! as HTMLSpanElement;

document.getElementById('update-open')!.addEventListener('click', () => {
    UpdateService.OpenReleasePage().catch(console.error);
});

UpdateService.Check().then((latest) => {
    if (latest) {
        updateBannerText.textContent = `Push Hack Installer ${latest} is available.`;
        updateBanner.hidden = false;
    }
}).catch(console.error);
