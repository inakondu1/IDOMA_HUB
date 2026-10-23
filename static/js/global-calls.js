(() => {
    "use strict";

    // The Messages page already has its own complete call interface.
    if (window.location.pathname === "/messages") return;

    let activeCall = null;
    let dismissedCallId = null;
    let checking = false;

    const style = document.createElement("style");
    style.textContent = `
        #idoma-global-call {
            position: fixed;
            right: 18px;
            bottom: 18px;
            z-index: 999999;
            width: min(360px, calc(100vw - 36px));
            padding: 20px;
            background: #111;
            color: #fff;
            border: 2px solid #b5121b;
            border-radius: 14px;
            box-shadow: 0 8px 30px rgba(0,0,0,.55);
            font-family: Arial, sans-serif;
        }
        #idoma-global-call[hidden] { display: none !important; }
        #idoma-global-call .igc-title {
            margin: 0 0 8px;
            font-size: 18px;
            font-weight: 700;
        }
        #idoma-global-call .igc-message {
            margin: 0 0 16px;
            color: #ddd;
            overflow-wrap: anywhere;
        }
        #idoma-global-call .igc-actions {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
        }
        #idoma-global-call button {
            border: 0;
            border-radius: 7px;
            padding: 10px 14px;
            cursor: pointer;
            font-weight: 700;
        }
        #idoma-global-call .igc-answer {
            background: #b5121b;
            color: white;
        }
        #idoma-global-call .igc-reject {
            background: #333;
            color: white;
        }
        #idoma-global-call .igc-dismiss {
            margin-left: auto;
            background: transparent;
            color: #ccc;
        }
    `;
    document.head.appendChild(style);

    const box = document.createElement("section");
    box.id = "idoma-global-call";
    box.hidden = true;
    box.setAttribute("role", "alertdialog");
    box.setAttribute("aria-live", "assertive");

    const title = document.createElement("h2");
    title.className = "igc-title";

    const message = document.createElement("p");
    message.className = "igc-message";

    const actions = document.createElement("div");
    actions.className = "igc-actions";

    const answer = document.createElement("button");
    answer.className = "igc-answer";
    answer.type = "button";
    answer.textContent = "Answer";

    const reject = document.createElement("button");
    reject.className = "igc-reject";
    reject.type = "button";
    reject.textContent = "Reject";

    const dismiss = document.createElement("button");
    dismiss.className = "igc-dismiss";
    dismiss.type = "button";
    dismiss.textContent = "Hide";

    actions.append(answer, reject, dismiss);
    box.append(title, message, actions);
    document.body.appendChild(box);

    function hidePopup() {
        box.hidden = true;
    }

    function showCall(call) {
        activeCall = call;
        const callerName = call.username || "Someone";
        const kind = call.call_type === "video" ? "video" : "voice";

        title.textContent = `Incoming ${kind} call`;
        message.textContent = `${callerName} is calling you.`;
        box.hidden = false;
    }

    async function sendAction(callId, actionName) {
        const body = new URLSearchParams({
            call_id: String(callId),
            action: actionName
        });

        const response = await fetch("/calls/action", {
            method: "POST",
            credentials: "same-origin",
            headers: {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            body: body.toString()
        });

        let data = {};
        try {
            data = await response.json();
        } catch (_) {}

        if (!response.ok) {
            throw new Error(data.error || "Unable to update the call.");
        }
    }

    answer.addEventListener("click", () => {
        if (!activeCall) return;
        const callerId = Number(activeCall.caller_id);
        if (!Number.isInteger(callerId) || callerId <= 0) return;

        // The Messages page owns the established audio/video call interface.
        const callId = Number(activeCall.call_id);
        if (!Number.isInteger(callId) || callId <= 0) return;

        window.location.href = "/messages?user_id=" +
            encodeURIComponent(String(callerId)) +
            "&answer_call=" + encodeURIComponent(String(callId));
    });

    reject.addEventListener("click", async () => {
        if (!activeCall) return;

        const call = activeCall;
        reject.disabled = true;

        try {
            await sendAction(call.call_id, "reject");
            dismissedCallId = Number(call.call_id);
            activeCall = null;
            hidePopup();
        } catch (error) {
            message.textContent = error.message || "Unable to reject the call.";
        } finally {
            reject.disabled = false;
        }
    });

    dismiss.addEventListener("click", () => {
        if (activeCall) dismissedCallId = Number(activeCall.call_id);
        hidePopup();
    });

    async function checkIncomingCall() {
        if (checking) return;
        checking = true;

        try {
            const response = await fetch("/calls/incoming-any", {
                method: "GET",
                credentials: "same-origin",
                cache: "no-store",
                headers: { "Accept": "application/json" }
            });

            if (!response.ok) return;

            const data = await response.json();
            const call = data.call;

            if (!call) {
                activeCall = null;
                dismissedCallId = null;
                hidePopup();
                return;
            }

            const callId = Number(call.call_id);
            if (!Number.isInteger(callId) || callId <= 0) return;

            if (dismissedCallId === callId) return;

            if (!activeCall || Number(activeCall.call_id) !== callId) {
                showCall(call);
            }
        } catch (_) {
            // A temporary network error should not break the page.
        } finally {
            checking = false;
        }
    }

    checkIncomingCall();
    window.setInterval(checkIncomingCall, 2500);
})();
