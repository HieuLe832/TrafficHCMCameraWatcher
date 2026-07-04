"use client";

import React, { useState, useEffect, useRef } from "react";

interface CameraPopupProps {
  id: string;
  name: string;
  code: string;
  status: string;
  onExpand?: () => void;
}

export default function CameraPopup({ id, name, code, status, onExpand }: CameraPopupProps) {
  const [loading, setLoading] = useState(true);
  const [timestamp, setTimestamp] = useState(Date.now());
  const [refreshInterval, setRefreshInterval] = useState<number>(7); // Auto-refresh 7s by default
  const timerRef = useRef<NodeJS.Timeout | null>(null);

  // Trigger a snapshot reload
  const handleRefresh = () => {
    setLoading(true);
    setTimestamp(Date.now());
  };

  // Manage auto-refresh interval
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

  // Format current local time for updated status
  const getFormattedTime = () => {
    const d = new Date(timestamp);
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
  };

  const imageUrl = `/api/camera/image?id=${id}&t=${timestamp}`;

  return (
    <div className="popup-content">
      <div className="popup-header">
        <div className="popup-title">{name}</div>
        <div className="popup-meta">
          <span className="camera-code">Mã: {code}</span>
          <span>Cập nhật: {getFormattedTime()}</span>
        </div>
      </div>

      <div className="popup-image-container">
        {loading && (
          <div className="popup-loader">
            <div className="spinner"></div>
            <span>Đang tải hình ảnh...</span>
          </div>
        )}
        {/* We use standard HTML img to load from proxy API */}
        <img
          className="popup-image"
          src={imageUrl}
          alt={name}
          onLoad={() => setLoading(false)}
          onError={() => setLoading(false)}
          key={timestamp}
        />
      </div>

      <div className="popup-footer">
        <button className="popup-btn popup-btn-icon" onClick={handleRefresh} title="Làm mới">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M23 4v6h-6M1 20v-6h6"></path>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
          </svg>
        </button>

        {onExpand && (
          <button className="popup-btn popup-btn-icon popup-btn-expand" onClick={onExpand} title="Phóng to">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="15 3 21 3 21 9"></polyline>
              <polyline points="9 21 3 21 3 15"></polyline>
              <line x1="21" y1="3" x2="14" y2="10"></line>
              <line x1="3" y1="21" x2="10" y2="14"></line>
            </svg>
          </button>
        )}

        <div className="popup-refresh-select">
          <span>Tự động:</span>
          <select
            className="popup-select"
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
      </div>
    </div>
  );
}
