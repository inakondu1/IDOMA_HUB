(() => {
    "use strict";

    const form = document.getElementById("chatMessageForm");
    const recordButton = document.getElementById("videoRecordButton");
    const stopButton = document.getElementById("videoStopButton");
    const cancelButton = document.getElementById("videoCancelButton");
    const status = document.getElementById("videoRecordingStatus");
    const preview = document.getElementById("videoPreview");

    if (!form || !recordButton || !stopButton || !cancelButton || !status || !preview) {
        return;
    }

    const MAX_VIDEO_BYTES = 10 * 1024 * 1024;

    let stream = null;
    let recorder = null;
    let chunks = [];
    let recordedBlob = null;
    let previewURL = null;
    let recording = false;
    let starting = false;
    let cancelRequested = false;
    let recordingFailed = false;

    function setStatus(message) {
        status.textContent = message;
    }

    function stopTracks() {
        if (stream) {
            stream.getTracks().forEach(track => track.stop());
            stream = null;
        }
    }

    function clearPreviewURL() {
        if (previewURL) {
            URL.revokeObjectURL(previewURL);
            previewURL = null;
        }
    }

    function resetControls() {
        recording = false;
        starting = false;
        recordButton.disabled = false;
        stopButton.disabled = true;
        cancelButton.disabled = !recordedBlob;
    }

    function clearRecording() {
        recordedBlob = null;
        chunks = [];
        preview.pause();
        preview.removeAttribute("src");
        preview.load();
        preview.hidden = true;
        clearPreviewURL();
        resetControls();
    }

    recordButton.addEventListener("click", async () => {
        if (recording || starting) return;

        if (!navigator.mediaDevices?.getUserMedia || !window.MediaRecorder) {
            setStatus("Video recording is not supported by this browser.");
            return;
        }

        starting = true;
        cancelRequested = false;
        recordingFailed = false;
        recordButton.disabled = true;
        cancelButton.disabled = true;
        setStatus("Requesting camera and microphone permission...");

        try {
            stream = await navigator.mediaDevices.getUserMedia({
                video: { facingMode: "user" },
                audio: true
            });

            const mimeType = [
                "video/webm;codecs=vp8,opus",
                "video/webm"
            ].find(type => MediaRecorder.isTypeSupported(type));

            recorder = mimeType
                ? new MediaRecorder(stream, { mimeType })
                : new MediaRecorder(stream);

            chunks = [];
            recordedBlob = null;

            recorder.addEventListener("dataavailable", event => {
                if (event.data && event.data.size > 0) {
                    chunks.push(event.data);
                }
            });

            recorder.addEventListener("stop", () => {
                stopTracks();

                if (recordingFailed) {
                    recordingFailed = false;
                    clearRecording();
                    setStatus("Recording failed. Please try again.");
                    return;
                }

                if (cancelRequested) {
                    cancelRequested = false;
                    clearRecording();
                    setStatus("Video recording cancelled.");
                    return;
                }

                if (chunks.length === 0) {
                    setStatus("No video was recorded. Please try again.");
                    clearRecording();
                    return;
                }

                recordedBlob = new Blob(chunks, {
                    type: recorder.mimeType || "video/webm"
                });

                if (recordedBlob.size > MAX_VIDEO_BYTES) {
                    clearRecording();
                    setStatus("This video exceeds 10 MB. Record a shorter video.");
                    return;
                }

                clearPreviewURL();
                previewURL = URL.createObjectURL(recordedBlob);
                preview.src = previewURL;
                preview.hidden = false;
                preview.load();

                recording = false;
                starting = false;
                recordButton.disabled = false;
                stopButton.disabled = true;
                cancelButton.disabled = false;
                setStatus("Video ready. Review it, then press Send to send it.");
            });

            recorder.addEventListener("error", () => {
                recordingFailed = true;
                stopTracks();
                clearRecording();
                setStatus("Recording failed. Please try again.");
            });

            recorder.start(500);

            recording = true;
            starting = false;
            recordButton.disabled = true;
            stopButton.disabled = false;
            cancelButton.disabled = false;
            setStatus("Recording video... Press Stop when finished.");
        } catch (error) {
            stopTracks();
            resetControls();
            setStatus(
                error.name === "NotAllowedError"
                    ? "Camera or microphone permission was denied."
                    : "Could not start the camera. Check your device and try again."
            );
        }
    });

    stopButton.addEventListener("click", () => {
        if (recorder && recorder.state !== "inactive") {
            stopButton.disabled = true;
            setStatus("Finishing video...");
            recorder.stop();
        }
    });

    cancelButton.addEventListener("click", () => {
        if (recorder && recorder.state !== "inactive") {
            cancelRequested = true;
            cancelButton.disabled = true;
            stopButton.disabled = true;
            setStatus("Cancelling video recording...");
            recorder.stop();
            return;
        }

        cancelRequested = false;
        clearRecording();
        setStatus("Video recording cancelled.");
    });

    form.addEventListener("submit", async event => {
        if (recording || starting) {
            event.preventDefault();
            setStatus("Stop recording before sending the video.");
            return;
        }

        if (!recordedBlob) return;

        event.preventDefault();

        const textarea = form.querySelector('textarea[name="content"]');
        if (textarea && textarea.value.trim()) {
            setStatus("Please clear the text field before sending a video message.");
            return;
        }

        const voiceControls = document.querySelector(".voice-controls");
        if (voiceControls?.querySelector("#voiceRecordingStatus")) {
            const voiceStatus = voiceControls.querySelector("#voiceRecordingStatus").textContent;
            if (/ready to send|recorded|sending voice/i.test(voiceStatus)) {
                setStatus("Please send or cancel the voice message first.");
                return;
            }
        }

        if (recordedBlob.size > MAX_VIDEO_BYTES) {
            setStatus("This video exceeds 10 MB. Record a shorter video.");
            return;
        }

        const sendButton = document.getElementById("sendMessageButton");
        if (sendButton) sendButton.disabled = true;
        recordButton.disabled = true;
        stopButton.disabled = true;
        cancelButton.disabled = true;
        setStatus("Uploading video message...");

        try {
            const data = new FormData();
            data.append("reply_to_id", form.querySelector('[name="reply_to_id"]')?.value || "0");
            data.append("video", recordedBlob, "video-message.webm");

            const response = await fetch(form.action, {
                method: "POST",
                body: data,
                credentials: "same-origin",
                redirect: "follow"
            });

            if (!response.ok) {
                throw new Error("Upload failed with status " + response.status);
            }

            window.location.assign(response.url || form.action);
        } catch (error) {
            console.error("Unable to send video message:", error);
            setStatus("Could not send the video. Check your connection and try again.");
            resetControls();
            if (sendButton) sendButton.disabled = false;
        }
    });

    window.addEventListener("beforeunload", () => {
        stopTracks();
        clearPreviewURL();
    });
})();
