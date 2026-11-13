import { useState } from 'react'
import { sizeText } from '../attachment'
import { CrossIcon } from '../icons'
import { useLanguage, useT } from '../i18n/useT'
import { attachmentURL, type ChatAttachment } from './chatApi'
import { Dialog } from './Dialog'

// Attachments are a message's files: scanning and rejected ones by name with their state, a clean image as its
// thumbnail that opens the picture in a preview, any other clean file as a download. The bytes are read from the
// API's own origin, built from the ids.
export function Attachments({ conversationId, files }: { conversationId: string; files: ChatAttachment[] }) {
  const t = useT()
  const { language } = useLanguage()
  const [open, setOpen] = useState<ChatAttachment>()
  return (
    <>
      <ul className="chat-files">
        {files.map((f) => (
          <li key={f.id}>
            {f.status === 'scanning' && <span className="muted">{t('chat.attachment.scanning', { name: f.filename })}</span>}
            {f.status === 'rejected' && <span className="danger-text">{t('chat.attachment.rejected', { name: f.filename })}</span>}
            {f.status === 'clean' && f.thumbnail && f.content_type.startsWith('image/') && (
              <button type="button" className="chat-thumb" aria-label={t('chat.attachment.open', { name: f.filename })} onClick={() => setOpen(f)}>
                <img src={attachmentURL(conversationId, f.id, 'thumbnail')} alt={f.filename} />
              </button>
            )}
            {f.status === 'clean' && !(f.thumbnail && f.content_type.startsWith('image/')) && (
              <a href={attachmentURL(conversationId, f.id, 'content')} download={f.filename}>
                {f.filename} <span className="muted">{sizeText(f.size, t, language)}</span>
              </a>
            )}
          </li>
        ))}
      </ul>
      {open && (
        <Dialog label={open.filename} className="viewer" onClose={() => setOpen(undefined)}>
          <div className="viewer-bar">
            <p className="viewer-title">
              <span className="file-title">{open.filename}</span>
            </p>
            <button type="button" className="icon" aria-label={t('chat.attachment.close')} title={t('chat.attachment.close')} onClick={() => setOpen(undefined)}>
              <CrossIcon />
            </button>
          </div>
          <div className="viewer-body">
            <img src={attachmentURL(conversationId, open.id, 'content')} alt={open.filename} />
          </div>
        </Dialog>
      )}
    </>
  )
}
