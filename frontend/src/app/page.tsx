"use client";

import React, { useState, useEffect, useMemo } from "react";
import dynamic from "next/dynamic";
import CameraViewer from "../components/CameraViewer";

interface Camera {
  id: string;
  code: string;
  name: string;
  lat: number;
  lng: number;
  status: string;
  angle: any;
  streaming: boolean;
  videoUrl?: string;
}

// Dynamically import Leaflet Map to avoid SSR errors in Next.js
const Map = dynamic(() => import("../components/Map"), {
  ssr: false,
  loading: () => (
    <div style={{
      width: "100%",
      height: "100%",
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      justifyContent: "center",
      backgroundColor: "#aad3df",
      color: "#333",
      gap: "12px",
      fontSize: "0.95rem",
      fontWeight: "500"
    }}>
      <div className="spinner" style={{ borderTopColor: "#2e7d32" }}></div>
      <span>Đang tải dữ liệu bản đồ...</span>
    </div>
  ),
});

export default function Home() {
  const [cameras, setCameras] = useState<Camera[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCamId, setSelectedCamId] = useState<string | null>(null);
  const [showOffline, setShowOffline] = useState(true);
  const [expandedCam, setExpandedCam] = useState<Camera | null>(null);

  // Fetch cameras from proxy server on mount
  useEffect(() => {
    fetch("/api/cameras")
      .then((res) => {
        if (!res.ok) {
          throw new Error(`Server returned HTTP status ${res.status}`);
        }
        return res.json();
      })
      .then((data: Camera[]) => {
        setCameras(data);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Fetch error:", err);
        setError(err.message || "Failed to load camera data");
        setLoading(false);
      });
  }, []);

  // Filter cameras based on search query and offline toggle
  const filteredCameras = useMemo(() => {
    return cameras.filter((cam) => {
      // Filter offline if unchecked
      if (!showOffline && cam.status !== "UP") {
        return false;
      }
      
      // Filter by search query (name or code)
      const q = searchQuery.toLowerCase();
      if (q) {
        const nameMatch = cam.name.toLowerCase().includes(q);
        const codeMatch = cam.code.toLowerCase().includes(q);
        return nameMatch || codeMatch;
      }
      
      return true;
    });
  }, [cameras, searchQuery, showOffline]);

  // Statistics
  const stats = useMemo(() => {
    const total = cameras.length;
    const online = cameras.filter((c) => c.status === "UP").length;
    return { total, online, filtered: filteredCameras.length };
  }, [cameras, filteredCameras]);

  const handleSelectCamera = (id: string | null) => {
    setSelectedCamId(id);
  };

  return (
    <main id="app-container">
      {/* LEFT PANEL SIDEBAR */}
      <section id="sidebar">
        <header id="sidebar-header">
          <h1>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"></path>
              <circle cx="12" cy="13" r="4"></circle>
            </svg>
            Camera Giao Thông HCMC
          </h1>
          <p>Hệ thống giám sát giao thông trực quan Thành phố Hồ Chí Minh</p>
        </header>

        {/* Search Panel */}
        <div id="search-container">
          <div className="search-wrapper">
            <input
              type="text"
              className="search-input"
              placeholder="Tìm theo tên đường hoặc mã camera..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            {searchQuery && (
              <button
                className="clear-search-btn"
                onClick={() => setSearchQuery("")}
                title="Xóa tìm kiếm"
              >
                ✕
              </button>
            )}
          </div>
        </div>

        {/* Filter Toggle */}
        <div className="filter-container">
          <label className="toggle-wrapper">
            <input
              type="checkbox"
              className="toggle-checkbox"
              checked={showOffline}
              onChange={(e) => setShowOffline(e.target.checked)}
            />
            <span>Hiển thị camera ngoại tuyến</span>
          </label>
        </div>

        {/* Sidebar list loader */}
        {loading ? (
          <div style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: "10px", color: "#666" }}>
            <div className="spinner" style={{ borderTopColor: "#2e7d32" }}></div>
            <span style={{ fontSize: "0.85rem" }}>Đang tải danh sách camera...</span>
          </div>
        ) : error ? (
          <div style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", padding: "20px", textAlign: "center", color: "#c62828", fontSize: "0.85rem" }}>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ marginBottom: "8px" }}>
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="12" y1="8" x2="12" y2="12"></line>
              <line x1="12" y1="16" x2="12.01" y2="16"></line>
            </svg>
            <strong>Lỗi tải dữ liệu:</strong>
            <p style={{ marginTop: "4px" }}>{error}</p>
          </div>
        ) : (
          <div className="camera-list">
            {filteredCameras.length === 0 ? (
              <div className="no-results">Không tìm thấy camera nào phù hợp</div>
            ) : (
              filteredCameras.map((cam) => (
                <div
                  key={cam.id}
                  className={`camera-item ${selectedCamId === cam.id ? "active" : ""}`}
                  onClick={() => handleSelectCamera(cam.id)}
                >
                  <div className="camera-name">{cam.name}</div>
                  <div className="camera-meta">
                    <span className="camera-status">
                      <span className={`status-dot ${cam.status === "UP" ? "up" : "down"}`} />
                      {cam.status === "UP" ? "Hoạt động" : "Ngoại tuyến"}
                    </span>
                    <span className="camera-code">{cam.code}</span>
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {/* Sidebar Footer Stats */}
        {!loading && !error && (
          <footer className="sidebar-stats">
            <span>Tổng số: <strong>{stats.total}</strong></span>
            <span>Trực tuyến: <strong style={{ color: "#2e7d32" }}>{stats.online}</strong></span>
            <span>Kết quả: <strong>{stats.filtered}</strong></span>
          </footer>
        )}
      </section>

      {/* RIGHT FULLSCREEN MAP */}
      <section id="map-container">
        {!loading && !error && (
          <Map
            cameras={cameras}
            selectedCamId={selectedCamId}
            onSelectCamera={handleSelectCamera}
            onExpandCamera={(cam) => setExpandedCam(cam)}
            showOffline={showOffline}
          />
        )}
      </section>

      {/* FULLSCREEN CAMERA VIEWER */}
      {expandedCam && (
        <CameraViewer
          id={expandedCam.id}
          name={expandedCam.name}
          code={expandedCam.code}
          status={expandedCam.status}
          onClose={() => setExpandedCam(null)}
        />
      )}
    </main>
  );
}
