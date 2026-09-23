// apps/artisan/src/lib/upload.ts
//
// One file straight to object storage, online only: upload-url, PUT,
// confirm. For the few uploads that are a live step in a flow (a literacy
// practice photo, an agent's voice-consent recording) rather than part of
// an offline listing draft, which goes through the outbox's media.upload.

import { generateUploadUrl, confirmUpload, type CallOptions } from '@kalakriti/api';

/** Returns the confirmed media id. */
export async function uploadFile(file: File, options?: CallOptions): Promise<string> {
  const contentType = file.type || 'application/octet-stream';
  const { media_id, upload_url } = await generateUploadUrl({ content_type: contentType, size_bytes: file.size }, options);
  if (!media_id || !upload_url) throw new Error('no upload url');
  const put = await fetch(upload_url, { method: 'PUT', body: file, headers: { 'Content-Type': contentType } });
  if (!put.ok) throw new Error(`upload failed: ${put.status}`);
  await confirmUpload(media_id, options);
  return media_id;
}
