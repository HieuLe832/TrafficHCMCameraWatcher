"use client";

import React, { useState, useEffect, useRef, useCallback } from "react";

interface CameraViewerProps {
  id: string;
  name: string;
  code: string;
  status: string;
  onClose: () => void;
}

export default function CameraViewer({ id, name, code, status, onClose }: CameraViewerProps) {
  const [loading, setLoading] = useState(true);
  const [timestamp, setTimestamp] = useState(Date.now());
  const [refreshInterval, setRefreshInterval] = useState<number>(7);
  const timerRef = useRef<NodeJS.Timeout | null>(null);
  const overlayRef = useRef<HTMLDivElement>(null);

  // Close on Escape key
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  // Close when clicking on overlay background
  const handleOverlayClick = useCallback((e: React.MouseEvent) => {
    if (e.target === overlayRef.current) {
      onClose();
    }
  }, [onClose]);

  // Auto-refresh interval
  useEffect(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }

    if (refreshInterval > 0) {
      timerRef.current = setInterval(() => {
        setTimestamp(Date.now());
      }, refreshInterval * 1000);
    }

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [refreshInterval]);

  const handleRefresh = () => {
    setLoading(true);
    setTimestamp(Date.now());
  };

  const getFormattedTime = () => {
    const d = new Date(timestamp);
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
  };

  const imageUrl = `/api/camera/image?id=${id}&t=${timestamp}`;

  return (
    <div className="viewer-overlay" ref={overlayRef} onClick={handleOverlayClick}>
      <div className="viewer-container">
        {/* Header */}
        <div className="viewer-header">
          <div className="viewer-header-left">
            <div className="viewer-title">{name}</div>
            <div className="viewer-meta">
              <span className="camera-code">Mã: {code}</span>
              <span className={`viewer-status ${status === "UP" ? "up" : "down"}`}>
                <span className={`status-dot ${status === "UP" ? "up" : "down"}`} />
                {status === "UP" ? "Hoạt động" : "Ngoại tuyến"}
              </span>
              <span>Cập nhật: {getFormattedTime()}</span>
            </div>
          </div>
          <button className="viewer-close-btn" onClick={onClose} title="Đóng (Esc)">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        {/* Image area */}
        <div className="viewer-image-area">
          {loading && (
            <div className="viewer-loader">
              <div className="spinner"></div>
              <span>Đang tải hình ảnh...</span>
            </div>
          )}
          <img
            className="viewer-image"
            src={imageUrl}
            alt={name}
            onLoad={() => setLoading(false)}
            onError={() => setLoading(false)}
            key={timestamp}
          />
        </div>

        {/* Footer controls */}
        <div className="viewer-footer">
          <button className="viewer-btn" onClick={handleRefresh}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M23 4v6h-6M1 20v-6h6"></path>
              <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
            </svg>
            Làm mới
          </button>

          <div className="viewer-refresh-select">
            <span>Tự động:</span>
            <select
              className="viewer-select"
              value={refreshInterval}
              onChange={(e) => setRefreshInterval(Number(e.target.value))}
            >
              <option value={0}>Không</option>
              <option value={5}>5 giây</option>
              <option value={7}>7 giây</option>
              <option value={10}>10 giây</option>
              <option value={15}>15 giây</option>
            </select>
          </div>

          <button className="viewer-btn viewer-btn-close" onClick={onClose}>
            Đóng
          </button>
        </div>
      </div>
    </div>
  );
}
