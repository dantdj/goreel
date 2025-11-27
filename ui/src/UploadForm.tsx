import { useState } from 'react';

export function UploadForm() {
    const [file, setFile] = useState<File | null>(null);
    const [uploading, setUploading] = useState(false);
    const [status, setStatus] = useState('');

    const handleUpload = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!file) return;

        setUploading(true);
        setStatus('Uploading...');

        const formData = new FormData();
        formData.append('video_file', file);

        try {
            const res = await fetch('http://127.0.0.1:8089/upload', {
                method: 'POST',
                body: formData,
            });

            if (!res.ok) {
                const text = await res.text();
                throw new Error(`Upload failed: ${res.status} ${text}`);
            }

            const data = await res.json();
            setStatus(`Success! Video ID: ${data.video_id}`);
        } catch (err: any) {
            setStatus(`Error: ${err.message}`);
        } finally {
            setUploading(false);
        }
    };

    return (
        <div className="card">
            <h2>Upload Video</h2>
            <form onSubmit={handleUpload} style={{ display: 'flex', flexDirection: 'column', gap: '1rem', alignItems: 'center' }}>
                <input
                    type="file"
                    accept="video/*"
                    onChange={(e) => setFile(e.target.files?.[0] || null)}
                />
                <button type="submit" disabled={!file || uploading}>
                    {uploading ? 'Uploading...' : 'Upload Video'}
                </button>
                {status && <p style={{ wordBreak: 'break-all' }}>{status}</p>}
            </form>
        </div>
    );
}
