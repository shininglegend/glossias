import { useAuthenticatedFetch } from "./authFetch";

export interface AudioUploadRequest {
  storyId: number;
  lineNumber: number;
  label: string;
  fileName: string;
}

export interface AudioUploadResponse {
  uploadUrl: string;
  filePath: string;
  fileBucket: string;
}

export interface AudioConfirmRequest {
  storyId: number;
  lineNumber: number;
  filePath: string;
  fileBucket: string;
  label: string;
}

export class AudioUploadError extends Error {
  constructor(
    message: string,
    public step: "request" | "upload" | "confirm",
  ) {
    super(message);
    this.name = "AudioUploadError";
  }
}

export function createAudioUploader() {
  const authenticatedFetch = useAuthenticatedFetch();

  return async function uploadAudioFile(
    file: File,
    storyId: number,
    lineNumber: number,
    label: string = "complete",
  ): Promise<void> {
    // Step 1: Request upload URL
    const uploadRequest: AudioUploadRequest = {
      storyId,
      lineNumber,
      label,
      fileName: file.name,
    };

    let uploadResponse: AudioUploadResponse;
    try {
      const response = await authenticatedFetch(
        "/api/admin/stories/audio/upload",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(uploadRequest),
        },
      );

      if (!response.ok) {
        throw new Error(`Failed to request upload URL: ${response.status}`);
      }

      uploadResponse = await response.json();
    } catch (error) {
      throw new AudioUploadError(
        error instanceof Error ? error.message : "Failed to request upload URL",
        "request",
      );
    }

    // Step 2: Upload file to signed URL
    try {
      const uploadFileResponse = await fetch(uploadResponse.uploadUrl, {
        method: "PUT",
        body: file,
        headers: {
          "Content-Type": file.type,
        },
      });

      if (!uploadFileResponse.ok) {
        throw new Error(`File upload failed: ${uploadFileResponse.status}`);
      }
    } catch (error) {
      throw new AudioUploadError(
        error instanceof Error ? error.message : "File upload failed",
        "upload",
      );
    }

    // Step 3: Confirm upload
    const confirmRequest: AudioConfirmRequest = {
      storyId,
      lineNumber,
      filePath: uploadResponse.filePath,
      fileBucket: uploadResponse.fileBucket,
      label,
    };

    try {
      const confirmResponse = await authenticatedFetch(
        "/api/admin/stories/audio/confirm",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(confirmRequest),
        },
      );

      if (!confirmResponse.ok) {
        throw new Error(`Failed to confirm upload: ${confirmResponse.status}`);
      }
    } catch (error) {
      throw new AudioUploadError(
        error instanceof Error ? error.message : "Failed to confirm upload",
        "confirm",
      );
    }
  };
}

export function createAudioDeleter() {
  const authenticatedFetch = useAuthenticatedFetch();

  return async function deleteLineAudio(
    storyId: number,
    lineNumber: number,
  ): Promise<void> {
    try {
      const response = await authenticatedFetch(
        "/api/admin/stories/audio/delete",
        {
          method: "DELETE",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ storyId, lineNumber }),
        },
      );

      if (!response.ok) {
        throw new Error(`Failed to delete audio: ${response.status}`);
      }
    } catch (error) {
      throw new AudioUploadError(
        error instanceof Error ? error.message : "Failed to delete audio",
        "request",
      );
    }
  };
}
