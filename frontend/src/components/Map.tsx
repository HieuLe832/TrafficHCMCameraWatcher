"use client";

import React, { useEffect } from "react";
import { MapContainer, TileLayer, Marker, Popup, useMap, ZoomControl } from "react-leaflet";
import L from "leaflet";
import MarkerClusterGroup from "react-leaflet-cluster";
import CameraPopup from "./CameraPopup";

interface Camera {
  id: string;
  code: string;
  name: string;
  lat: float64;
  lng: float64;
  status: string;
  angle: any;
  streaming: boolean;
  videoUrl?: string;
}

// In TS, float64 maps to number
type float64 = number;

interface MapProps {
  cameras: Camera[];
  selectedCamId: string | null;
  onSelectCamera: (id: string | null) => void;
  onExpandCamera: (cam: Camera) => void;
  showOffline: boolean;
}

// Custom Marker Icon builder
const createCameraIcon = (status: string, angle: number | null, isActive: boolean) => {
  const isUp = status === "UP";
  const colorClass = isUp ? "up" : "down";
  const activeClass = isActive ? "active" : "";
  
  // Custom arrow element for camera rotation
  const rotationStyle = angle !== null ? `transform: rotate(${angle}deg);` : "display: none;";
  const strokeColor = isUp ? "#2e7d32" : "#c62828";

  return L.divIcon({
    className: "camera-marker-icon",
    html: `
      <div class="marker-pin-wrapper">
        <div class="marker-arrow" style="${rotationStyle}"></div>
        <div class="marker-pin ${colorClass} ${activeClass}">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="${strokeColor}" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"></path>
            <circle cx="12" cy="13" r="4"></circle>
          </svg>
        </div>
      </div>
    `,
    iconSize: [38, 38],
    iconAnchor: [19, 19],
    popupAnchor: [0, -19]
  });
};

// Custom Cluster Icon builder
const createClusterCustomIcon = (cluster: any) => {
  const count = cluster.getChildCount();
  let size = 36;
  if (count >= 50) {
    size = 48;
  } else if (count >= 10) {
    size = 42;
  }
  return L.divIcon({
    html: `
      <div class="custom-cluster-marker" style="width: ${size}px; height: ${size}px; line-height: ${size - 4}px;">
        <span>${count}</span>
      </div>
    `,
    className: "custom-camera-cluster",
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2]
  });
};

// Map controller to handle programmatical pans (flyTo)
function MapController({ selectedCam }: { selectedCam: Camera | null }) {
  const map = useMap();

  useEffect(() => {
    if (selectedCam) {
      map.flyTo([selectedCam.lat, selectedCam.lng], 17, {
        animate: true,
        duration: 1.5,
      });
    }
  }, [selectedCam, map]);

  return null;
}

export default function Map({ cameras, selectedCamId, onSelectCamera, onExpandCamera, showOffline }: MapProps) {
  // Find selected camera details
  const selectedCam = cameras.find((c) => c.id === selectedCamId) || null;

  // Filter cameras
  const filteredCameras = cameras.filter((cam) => {
    if (!showOffline && cam.status !== "UP") return false;
    return true;
  });

  return (
    <div style={{ width: "100%", height: "100%" }}>
      <MapContainer
        center={[10.7769, 106.7009]} // Ho Chi Minh City center coordinates
        zoom={14}
        scrollWheelZoom={true}
        zoomControl={false} // Disable default zoom control to customize its position later if needed
      >
        {/* We use CartoDB Positron for a cleaner, simpler map background */}
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>'
          url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
        />

        {/* Dynamic flyTo handler */}
        <MapController selectedCam={selectedCam} />

        {/* Custom Zoom Control placed at top-right for premium layout */}
        <ZoomControl position="topright" />

        {/* Camera markers grouped in clusters */}
        <MarkerClusterGroup
          chunkedLoading
          iconCreateFunction={createClusterCustomIcon}
          maxClusterRadius={50}
          disableClusteringAtZoom={16}
        >
          {filteredCameras.map((cam) => {
            const isActive = cam.id === selectedCamId;
            const parsedAngle = cam.angle !== null && cam.angle !== undefined ? Number(cam.angle) : null;

            return (
              <Marker
                key={cam.id}
                position={[cam.lat, cam.lng]}
                icon={createCameraIcon(cam.status, parsedAngle, isActive)}
                eventHandlers={{
                  click: () => {
                    onSelectCamera(cam.id);
                  },
                  popupclose: () => {
                    onSelectCamera(null);
                  }
                }}
              >
                <Popup className="custom-popup" maxWidth={300} minWidth={300}>
                  <CameraPopup
                    id={cam.id}
                    name={cam.name}
                    code={cam.code}
                    status={cam.status}
                    onExpand={() => onExpandCamera(cam)}
                  />
                </Popup>
              </Marker>
            );
          })}
        </MarkerClusterGroup>
      </MapContainer>
    </div>
  );
}
