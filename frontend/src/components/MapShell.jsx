import React, { useEffect, useRef, useState } from 'react';
import gameData from '../mock/game-data.json';

const OSM_CENTER = [39.908722, 116.397499];

function loadLeaflet() {
  if (window.L) return Promise.resolve(window.L);

  return new Promise((resolve, reject) => {
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
    document.head.appendChild(link);

    const script = document.createElement('script');
    script.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
    script.async = true;
    script.onload = () => resolve(window.L);
    script.onerror = () => reject(new Error('Failed to load Leaflet'));
    document.body.appendChild(script);
  });
}

export default function MapShell({ onOpenPortal }) {
  const mapRef = useRef(null);
  const containerRef = useRef(null);
  const overlayRef = useRef({
    portals: null,
    links: null,
    fields: null,
    player: null,
    path: null,
    arrow: null
  });
  const animRef = useRef({ raf: null, bearing: 0 });
  const [mapError, setMapError] = useState('');
  const [playerPos, setPlayerPos] = useState(OSM_CENTER);

  useEffect(() => {
    let isMounted = true;

    loadLeaflet()
      .then((L) => {
        if (!isMounted || mapRef.current) return;
        mapRef.current = L.map(containerRef.current, {
          center: OSM_CENTER,
          zoom: 16,
          zoomControl: true
        });
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
          maxZoom: 19,
          attribution: '&copy; OpenStreetMap contributors'
        }).addTo(mapRef.current);

        const portals = gameData.portals || [];
        const links = gameData.links || [];
        const fields = gameData.fields || [];

        const fieldLayer = L.layerGroup().addTo(mapRef.current);
        fields.forEach((field) => {
          const points = (field.points || []).map((pt) => [pt.lat, pt.lng]);
          const style = styleForFaction(field.faction);
          if (points.length >= 3) {
            L.polygon(points, {
              color: style.stroke,
              weight: 2,
              fillColor: style.fill,
              fillOpacity: 0.18
            }).addTo(fieldLayer);
          }
        });

        const linkLayer = L.layerGroup().addTo(mapRef.current);
        links.forEach((link) => {
          const style = styleForFaction(link.faction);
          L.polyline(
            [
              [link.from.lat, link.from.lng],
              [link.to.lat, link.to.lng]
            ],
            {
              color: style.stroke,
              weight: 2,
              opacity: 0.9
            }
          ).addTo(linkLayer);
        });

        const portalLayer = L.layerGroup().addTo(mapRef.current);
        portals.forEach((portal) => {
          const factionClass = factionToClass(portal.faction);
          const icon = L.divIcon({
            className: 'portal-marker',
            html: `<div class=\"portal ${factionClass}\">L${portal.level || 0}</div>`,
            iconSize: [46, 46],
            iconAnchor: [23, 23]
          });
          const marker = L.marker([portal.lat, portal.lng], { icon }).addTo(portalLayer);
          marker.on('click', () => onOpenPortal(portal));
        });

        const playerIcon = L.divIcon({
          className: 'player-marker',
          html: '<div class="player-nav"><span></span></div>',
          iconSize: [32, 32],
          iconAnchor: [16, 16]
        });
        const playerMarker = L.marker(playerPos, { icon: playerIcon }).addTo(mapRef.current);
        const playerArrow = L.marker(playerPos, {
          icon: L.divIcon({
            className: 'player-direction',
            html: '<div class="player-direction-nav"><span></span></div>',
            iconSize: [26, 26],
            iconAnchor: [13, 13]
          }),
          interactive: false
        }).addTo(mapRef.current);

        const applyBearing = (bearing) => {
          const markerEl = playerMarker.getElement();
          if (markerEl) {
            const nav = markerEl.querySelector('.player-nav');
            if (nav) {
              nav.style.setProperty('--bearing', `${bearing}deg`);
            }
          }
          const dirEl = playerArrow.getElement();
          if (dirEl) {
            const nav = dirEl.querySelector('.player-direction-nav');
            if (nav) {
              nav.style.setProperty('--bearing', `${bearing}deg`);
            }
          }
        };

        const movePlayerTo = (nextPos) => {
          const startPos = playerMarker.getLatLng();
          const start = [startPos.lat, startPos.lng];
          const distance = haversineMeters(start, nextPos);
          const duration = Math.max(distance / 20, 0.5) * 1000;
          const targetBearing = bearingDeg(start, nextPos);
          const current = animRef.current.bearing || 0;
          const delta = shortestDeltaDeg(current, targetBearing);
          const nextBearing = current + delta;
          animRef.current.bearing = nextBearing;
          applyBearing(nextBearing);

          if (overlayRef.current.path) {
            overlayRef.current.path.remove();
          }
          if (overlayRef.current.arrow) {
            overlayRef.current.arrow.remove();
          }

          overlayRef.current.path = L.polyline([start, nextPos], {
            color: '#7fc9ff',
            dashArray: '4 6',
            weight: 3,
            opacity: 0.95
          }).addTo(mapRef.current);

          overlayRef.current.arrow = L.marker(nextPos, {
            icon: L.divIcon({
              className: 'path-arrow',
              html: '<div class="path-arrow-head"></div>',
              iconSize: [14, 14],
              iconAnchor: [7, 7]
            }),
            interactive: false
          }).addTo(mapRef.current);

          const arrowEl2 = overlayRef.current.arrow.getElement();
          if (arrowEl2) {
            arrowEl2.style.transform = `rotate(${targetBearing}deg)`;
          }

          if (animRef.current.raf) {
            cancelAnimationFrame(animRef.current.raf);
          }

          const startTime = performance.now();
          const animate = (now) => {
            const t = Math.min((now - startTime) / duration, 1);
            const lat = start[0] + (nextPos[0] - start[0]) * t;
            const lng = start[1] + (nextPos[1] - start[1]) * t;
            playerMarker.setLatLng([lat, lng]);
            playerArrow.setLatLng([lat, lng]);
            if (t < 1) {
              animRef.current.raf = requestAnimationFrame(animate);
            } else {
              setPlayerPos(nextPos);
            }
          };

          animRef.current.raf = requestAnimationFrame(animate);
        };

        mapRef.current.on('contextmenu', (event) => {
          const nextPos = [event.latlng.lat, event.latlng.lng];
          movePlayerTo(nextPos);
        });

        overlayRef.current = {
          portals: portalLayer,
          links: linkLayer,
          fields: fieldLayer,
          player: playerMarker,
          path: null,
          arrow: playerArrow
        };
      })
      .catch((err) => {
        if (isMounted) {
          setMapError(err.message || 'Leaflet load failed');
        }
      });

    return () => {
      isMounted = false;
      if (mapRef.current) {
        if (overlayRef.current.portals) overlayRef.current.portals.remove();
        if (overlayRef.current.links) overlayRef.current.links.remove();
        if (overlayRef.current.fields) overlayRef.current.fields.remove();
        if (overlayRef.current.player) overlayRef.current.player.remove();
        if (overlayRef.current.path) overlayRef.current.path.remove();
        if (overlayRef.current.arrow) overlayRef.current.arrow.remove();
        overlayRef.current = { portals: null, links: null, fields: null, player: null, path: null, arrow: null };
        if (animRef.current.raf) {
          cancelAnimationFrame(animRef.current.raf);
          animRef.current.raf = null;
        }
        mapRef.current.remove();
        mapRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    const handleKeyDown = (event) => {
      if (event.key === 'h' || event.key === 'H') {
        if (mapRef.current) {
          mapRef.current.setView(playerPos, mapRef.current.getZoom());
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [playerPos]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return undefined;
    const preventContextMenu = (event) => event.preventDefault();
    container.addEventListener('contextmenu', preventContextMenu);
    return () => {
      container.removeEventListener('contextmenu', preventContextMenu);
    };
  }, []);

  return (
    <section className="map-canvas" id="map">
      <div className="map-container" ref={containerRef} />
      {mapError ? <div className="map-error">{mapError}</div> : null}
    </section>
  );
}

function styleForFaction(faction) {
  switch (faction) {
    case 'RESISTANCE':
      return { stroke: '#3aa0ff', fill: '#3aa0ff' };
    case 'ENLIGHTENED':
      return { stroke: '#3cff8d', fill: '#3cff8d' };
    case 'NEUTRAL':
      return { stroke: '#9fb0c7', fill: '#9fb0c7' };
    default:
      return { stroke: '#ff4d4d', fill: '#ff4d4d' };
  }
}

function factionToClass(faction) {
  switch (faction) {
    case 'RESISTANCE':
      return 'res';
    case 'ENLIGHTENED':
      return 'enl';
    case 'NEUTRAL':
      return 'neutral';
    default:
      return 'red';
  }
}

function haversineMeters(a, b) {
  const toRad = (d) => (d * Math.PI) / 180;
  const lat1 = toRad(a[0]);
  const lat2 = toRad(b[0]);
  const dLat = lat2 - lat1;
  const dLng = toRad(b[1] - a[1]);
  const sinLat = Math.sin(dLat / 2);
  const sinLng = Math.sin(dLng / 2);
  const h = sinLat * sinLat + Math.cos(lat1) * Math.cos(lat2) * sinLng * sinLng;
  return 6371000 * 2 * Math.asin(Math.min(1, Math.sqrt(h)));
}

function bearingDeg(a, b) {
  const toRad = (d) => (d * Math.PI) / 180;
  const toDeg = (r) => (r * 180) / Math.PI;
  const lat1 = toRad(a[0]);
  const lat2 = toRad(b[0]);
  const dLng = toRad(b[1] - a[1]);
  const y = Math.sin(dLng) * Math.cos(lat2);
  const x = Math.cos(lat1) * Math.sin(lat2) - Math.sin(lat1) * Math.cos(lat2) * Math.cos(dLng);
  const brng = toDeg(Math.atan2(y, x));
  return (brng + 360) % 360;
}

function shortestDeltaDeg(current, target) {
  const diff = ((target - current + 540) % 360) - 180;
  return diff;
}
