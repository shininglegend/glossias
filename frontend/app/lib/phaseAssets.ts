import { useCallback } from "react";
import { useAuthenticatedFetch } from "./authFetch";
import type { PhaseAssetKind } from "../types/admin";

// Uploading a picture or word audio for the Summer 2026 phases is the same
// three-step flow as line audio (see lib/audio.ts): ask for a signed URL, PUT
// the bytes straight to storage, then confirm. The difference is the confirm
// step — instead of registering a story_images row, the caller attaches the
// returned path to the target word or recall sentence that owns it, via that
// editor's save endpoint. Those columns are the source of truth for which asset
// belongs to which row.

export interface PhaseAssetUploadResponse {
  uploadUrl: string;
  filePath: string;
  fileBucket: string;
}

export class PhaseAssetUploadError extends Error {
  constructor(
    message: string,
    public step: "request" | "upload",
  ) {
    super(message);
    this.name = "PhaseAssetUploadError";
  }
}

/**
 * Uploads one asset and returns the storage path to attach to its owning row.
 * The asset is not referenced by anything until that attach call succeeds.
 */
export function usePhaseAssetUploader() {
  const authenticatedFetch = useAuthenticatedFetch();

  return useCallback(
    async function uploadPhaseAsset(
      file: File,
      storyId: number,
      kind: PhaseAssetKind,
      ownerId: number,
    ): Promise<string> {
      let minted: PhaseAssetUploadResponse;
      try {
        const response = await authenticatedFetch(
          `/api/admin/stories/${storyId}/phase-assets/upload`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ kind, ownerId, fileName: file.name }),
          },
        );

        if (!response.ok) {
          throw new Error((await response.text()) || `HTTP ${response.status}`);
        }

        minted = await response.json();
      } catch (error) {
        throw new PhaseAssetUploadError(
          error instanceof Error
            ? error.message
            : "Failed to request upload URL",
          "request",
        );
      }

      try {
        const uploaded = await fetch(minted.uploadUrl, {
          method: "PUT",
          body: file,
          headers: { "Content-Type": file.type },
        });

        if (!uploaded.ok) {
          throw new Error(`File upload failed: ${uploaded.status}`);
        }
      } catch (error) {
        throw new PhaseAssetUploadError(
          error instanceof Error ? error.message : "File upload failed",
          "upload",
        );
      }

      return minted.filePath;
    },
    [authenticatedFetch],
  );
}
