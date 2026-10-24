(() => {
    const form = document.getElementById("chatMessageForm");
    const textBox = form?.querySelector('textarea[name="content"]');
    const recordButton = document.getElementById("voiceRecordButton");
    const stopButton = document.getElementById("voiceStopButton");
    const cancelButton = document.getElementById("voiceCancelButton");
    const sendButton = document.getElementById("sendMessageButton");
    const status = document.getElementById("voiceRecordingStatus");

    if (!form || !textBox || !recordButton || !stopButton ||
        !cancelButton || !sendButton || !status) {
        return;
    }

    let recorder = null;
    let microphone = null;
    let chunks = [];
    let recordingBlob = null;
    let recordingType = "";

    function releaseMicrophone() {
        if (microphone) {
            microphone.getTracks().forEach(track => track.stop());
            microphone = null;
        }
    }

    function resetRecording() {
        recordingBlob = null;
        recordingType = "";
        chunks = [];
        recordButton.disabled = false;
        stopButton.disabled = true;
        cancelButton.disabled = true;
        sendButton.disabled = false;
    }

    recordButton.addEventListener("click", async () => {
        if (!navigator.mediaDevices?.getUserMedia ||
            typeof MediaRecorder === "undefined") {
            status.textContent =
                "Voice recording is not supported by this browser.";
            return;
        }

        if (textBox.value.trim()) {
            status.textContent =
                "Send your text first, then record a voice message.";
            return;
        }

        try {
            microphone = await navigator.mediaDevices.getUserMedia({
                audio: true
            });

            const preferredType = [
                "audio/webm;codecs=opus",
                "audio/webm",
                "audio/mp4"
            ].find(type => MediaRecorder.isTypeSupported(type));

            recorder = preferredType
                ? new MediaRecorder(microphone, { mimeType: preferredType })
                : new MediaRecorder(microphone);

            chunks = [];
            recordingBlob = null;

            recorder.addEventListener("dataavailable", event => {
                if (event.data && event.data.size > 0) {
                    chunks.push(event.data);
                }
            });

            recorder.addEventListener("stop", () => {
                if (chunks.length) {
                    recordingType = recorder.mimeType || "audio/webm";
                    recordingBlob = new Blob(chunks, {
                        type: recordingType
                    });

                    if (recordingBlob.size > 10 * 1024 * 1024) {
                        recordingBlob = null;
                        status.textContent =
                            "Recording exceeds 10 MB. Please record a shorter message.";
                        resetRecording();
                    } else if (recordingBlob.size === 0) {
                        status.textContent = "No audio was recorded. Try again.";
                        resetRecording();
                    } else {
                        status.textContent =
                            "Voice note ready. Press Send to deliver it.";
                        recordButton.disabled = true;
                        stopButton.disabled = true;
                        cancelButton.disabled = false;
                        sendButton.disabled = false;
                    }
                }

                releaseMicrophone();
            });

            recorder.start();
            recordButton.disabled = true;
            stopButton.disabled = false;
            cancelButton.disabled = false;
            sendButton.disabled = true;
            status.textContent = "Recording… Press Stop when finished.";
        } catch (error) {
            releaseMicrophone();
            status.textContent =
                "Could not access the microphone. Check browser permissions and try again.";
            console.error("Voice recording error:", error);
        }
    });

    stopButton.addEventListener("click", () => {
        if (recorder && recorder.state === "recording") {
            recorder.stop();
            status.textContent = "Preparing your voice note…";
            stopButton.disabled = true;
        }
    });

    cancelButton.addEventListener("click", () => {
        if (recorder && recorder.state !== "inactive") {
            recorder.addEventListener("stop", () => {
                resetRecording();
                status.textContent = "Recording cancelled.";
            }, { once: true });
            recorder.stop();
        } else {
            resetRecording();
            releaseMicrophone();
            status.textContent = "Voice note cancelled.";
        }
    });

    form.addEventListener("submit", event => {
        if (!recordingBlob) {
            return;
        }

        event.preventDefault();

        if (recordingBlob.size > 10 * 1024 * 1024) {
            status.textContent = "Voice message must be 10 MB or smaller.";
            return;
        }

        const extension = recordingType.includes("mp4") ? "m4a" : "webm";
        const payload = new FormData();
        payload.append("voice", recordingBlob, `voice-message.${extension}`);

        sendButton.disabled = true;
        recordButton.disabled = true;
        status.textContent = "Sending voice message…";

        fetch(form.action, {
            method: "POST",
            body: payload,
            credentials: "same-origin",
            redirect: "follow"
        })
        .then(response => {
            if (!response.ok) {
                throw new Error(`Upload failed (${response.status})`);
            }

            window.location.href = response.url || form.action;
        })
        .catch(error => {
            console.error("Voice upload error:", error);
            status.textContent =
                "Could not send the voice note. Please try again.";
            sendButton.disabled = false;
            recordButton.disabled = false;
        });
    });

    window.addEventListener("pagehide", releaseMicrophone);
})();
