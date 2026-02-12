import React, { useEffect, useRef, useState } from 'react';

const OSM_CENTER = [39.908722, 116.397499];
const ACTION_RADIUS_METERS = 40;
const HEADING_SPEED_THRESHOLD_MPS = 0.15;
const INITIAL_FOCUS_ZOOM = 18;
const SNAP_DISTANCE_METERS = 0.35;
const FOLLOW_EASE_MS = 190;
const FOLLOW_MIN_ALPHA = 0.08;
const FOLLOW_MAX_ALPHA = 0.45;
const TELEPORT_SNAP_METERS = 120;
const MOTION_GRACE_MS = 1200;

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

export default function MapShell({
  portals = [],
  links = [],
  fields = [],
  player,
  moveTarget,
  attackPulses = [],
  attackChargeFx = null,
  nearbyPlayers = [],
  onOpenPortal,
  onTargetUpdate,
  onViewUpdate,
  linkMode,
  onCancelLinkMode
}) {
  const mapRef = useRef(null);
  const leafletRef = useRef(null);
  const containerRef = useRef(null);
  const overlayRef = useRef({
    portals: null,
    links: null,
    fields: null,
    charge: null,
    pulses: null,
    nearby: null,
    player: null,
    operationRange: null,
    preRender: null,
    target: null,
    path: null,
    preRenderPath: null,
    arrow: null
  });
  const playerPosRef = useRef([
    Number(player?.renderLatitude ?? player?.latitude ?? OSM_CENTER[0]),
    Number(player?.renderLongitude ?? player?.longitude ?? OSM_CENTER[1])
  ]);
  const animRef = useRef({
    raf: null,
    bearing: 0,
    lastRender: null,
    displayPos: null,
    desiredPos: null,
    lastFrameTs: 0,
    motion: null
  });
  const initialFocusDoneRef = useRef(false);
  const lastServerTargetKeyRef = useRef('');
  const applyBearingRef = useRef(() => {});
  const [mapError, setMapError] = useState('');
  const [mapReady, setMapReady] = useState(false);
  const [pendingTarget, setPendingTarget] = useState(null);

  useEffect(() => {
    let isMounted = true;

    loadLeaflet()
      .then((L) => {
        if (!isMounted || mapRef.current) return;
        leafletRef.current = L;
        mapRef.current = L.map(containerRef.current, {
          center: playerPosRef.current,
          zoom: INITIAL_FOCUS_ZOOM,
          zoomControl: true
        });
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
          maxZoom: 19,
          attribution: '&copy; OpenStreetMap contributors'
        }).addTo(mapRef.current);

        const fieldLayer = L.layerGroup().addTo(mapRef.current);
        const linkLayer = L.layerGroup().addTo(mapRef.current);
        const portalLayer = L.layerGroup().addTo(mapRef.current);
        const chargeLayer = L.layerGroup().addTo(mapRef.current);
        const pulseLayer = L.layerGroup().addTo(mapRef.current);
        const nearbyLayer = L.layerGroup().addTo(mapRef.current);

        const playerIcon = L.divIcon({
          className: 'player-marker',
          html: '<div class="player-nav"><span></span></div>',
          iconSize: [32, 32],
          iconAnchor: [16, 16]
        });
        const playerMarker = L.marker(playerPosRef.current, { icon: playerIcon }).addTo(mapRef.current);
        const playerArrow = L.marker(playerPosRef.current, {
          icon: L.divIcon({
            className: 'player-direction',
            html: '<div class="player-direction-nav"><span></span></div>',
            iconSize: [26, 26],
            iconAnchor: [13, 13]
          }),
          interactive: false
        }).addTo(mapRef.current);
        const preRenderMarker = L.circleMarker(playerPosRef.current, {
          radius: 4,
          color: '#7fc9ff',
          weight: 1,
          fillColor: '#7fc9ff',
          fillOpacity: 0.35,
          className: 'player-pre-render-marker',
          interactive: false
        }).addTo(mapRef.current);
        const operationRange = L.circle(playerPosRef.current, {
          radius: ACTION_RADIUS_METERS,
          color: '#f6c356',
          weight: 2,
          fill: false,
          interactive: false
        }).addTo(mapRef.current);

        const applyBearing = (bearing) => {
          const markerEl = playerMarker.getElement();
          if (markerEl) {
            const nav = markerEl.querySelector('.player-nav');
            if (nav) nav.style.setProperty('--bearing', `${bearing}deg`);
          }
          const dirEl = playerArrow.getElement();
          if (dirEl) {
            const nav = dirEl.querySelector('.player-direction-nav');
            if (nav) nav.style.setProperty('--bearing', `${bearing}deg`);
          }
        };
        applyBearingRef.current = applyBearing;

        const emitBounds = () => {
          if (!mapRef.current) return;
          const bounds = mapRef.current.getBounds();
          onViewUpdate?.({
            minLat: bounds.getSouth(),
            maxLat: bounds.getNorth(),
            minLon: bounds.getWest(),
            maxLon: bounds.getEast()
          });
        };

        mapRef.current.on('contextmenu', (event) => {
          const nextPos = [event.latlng.lat, event.latlng.lng];
          setPendingTarget(nextPos);
          onTargetUpdate?.({ latitude: nextPos[0], longitude: nextPos[1] });
        });

        mapRef.current.on('moveend', emitBounds);
        mapRef.current.on('zoomend', emitBounds);

        overlayRef.current = {
          portals: portalLayer,
          links: linkLayer,
          fields: fieldLayer,
          charge: chargeLayer,
          pulses: pulseLayer,
          nearby: nearbyLayer,
          player: playerMarker,
          operationRange,
          preRender: preRenderMarker,
          target: null,
          path: null,
          preRenderPath: null,
          arrow: playerArrow
        };
        setMapReady(true);
        setTimeout(emitBounds, 0);
      })
      .catch((err) => {
        if (isMounted) setMapError(err.message || 'Leaflet load failed');
      });

    return () => {
      isMounted = false;
      if (mapRef.current) {
        if (overlayRef.current.portals) overlayRef.current.portals.remove();
        if (overlayRef.current.links) overlayRef.current.links.remove();
        if (overlayRef.current.fields) overlayRef.current.fields.remove();
        if (overlayRef.current.charge) overlayRef.current.charge.remove();
        if (overlayRef.current.pulses) overlayRef.current.pulses.remove();
        if (overlayRef.current.nearby) overlayRef.current.nearby.remove();
        if (overlayRef.current.player) overlayRef.current.player.remove();
        if (overlayRef.current.operationRange) overlayRef.current.operationRange.remove();
        if (overlayRef.current.preRender) overlayRef.current.preRender.remove();
        if (overlayRef.current.target) overlayRef.current.target.remove();
        if (overlayRef.current.path) overlayRef.current.path.remove();
        if (overlayRef.current.preRenderPath) overlayRef.current.preRenderPath.remove();
        if (overlayRef.current.arrow) overlayRef.current.arrow.remove();
        if (animRef.current.raf) cancelAnimationFrame(animRef.current.raf);
        mapRef.current.remove();
        mapRef.current = null;
      }
      applyBearingRef.current = () => {};
    };
  }, []);

  useEffect(() => {
    const serverTarget = [
      Number(moveTarget?.latitude ?? moveTarget?.lat),
      Number(moveTarget?.longitude ?? moveTarget?.lng)
    ];
    const hasServerTarget = Number.isFinite(serverTarget[0]) && Number.isFinite(serverTarget[1]);
    const nextKey = hasServerTarget ? `${serverTarget[0].toFixed(7)}:${serverTarget[1].toFixed(7)}` : '';
    if (nextKey === lastServerTargetKeyRef.current) return;
    lastServerTargetKeyRef.current = nextKey;
    setPendingTarget(hasServerTarget ? serverTarget : null);
  }, [moveTarget?.lat, moveTarget?.latitude, moveTarget?.lng, moveTarget?.longitude]);

  useEffect(() => {
    const map = mapRef.current;
    if (!mapReady || !map || initialFocusDoneRef.current) return;
    const render = [
      Number(player?.renderLatitude ?? player?.latitude),
      Number(player?.renderLongitude ?? player?.longitude)
    ];
    if (!Number.isFinite(render[0]) || !Number.isFinite(render[1])) return;
    map.setView(render, Math.max(map.getZoom(), INITIAL_FOCUS_ZOOM), { animate: false });
    playerPosRef.current = render;
    initialFocusDoneRef.current = true;

    const bounds = map.getBounds();
    onViewUpdate?.({
      minLat: bounds.getSouth(),
      maxLat: bounds.getNorth(),
      minLon: bounds.getWest(),
      maxLon: bounds.getEast()
    });
  }, [mapReady, onViewUpdate, player?.latitude, player?.longitude, player?.renderLatitude, player?.renderLongitude]);

  useEffect(() => {
    const marker = overlayRef.current.player;
    const operationRange = overlayRef.current.operationRange;
    const preRenderMarker = overlayRef.current.preRender;
    const arrow = overlayRef.current.arrow;
    if (!mapReady || !marker || !arrow || !preRenderMarker || !operationRange) return;

    const render = [
      Number(player?.renderLatitude ?? player?.latitude),
      Number(player?.renderLongitude ?? player?.longitude)
    ];
    if (!Number.isFinite(render[0]) || !Number.isFinite(render[1])) return;

    const preRender = [
      Number(player?.preRenderLatitude ?? render[0]),
      Number(player?.preRenderLongitude ?? render[1])
    ];
    preRenderMarker.setLatLng(preRender);

    const speedMps = Number(player?.speedMps);
    const headingFromPayload = Number(player?.headingDeg);
    const hasPreRenderDelta = haversineMeters(render, preRender) > SNAP_DISTANCE_METERS;
    const movingBySpeed = Number.isFinite(speedMps) && speedMps > HEADING_SPEED_THRESHOLD_MPS;
    const hasMotionSignal = hasPreRenderDelta || movingBySpeed;
    const serverDesiredPos = hasMotionSignal ? preRender : render;
    const routeTarget = [
      Number(moveTarget?.latitude ?? moveTarget?.lat),
      Number(moveTarget?.longitude ?? moveTarget?.lng)
    ];
    const hasRouteTarget = Number.isFinite(routeTarget[0]) && Number.isFinite(routeTarget[1]);

    const setDisplayPos = (nextPos) => {
      const currentMarker = overlayRef.current.player;
      const currentArrow = overlayRef.current.arrow;
      const currentRange = overlayRef.current.operationRange;
      if (!currentMarker || !currentArrow || !currentRange) return;
      currentMarker.setLatLng(nextPos);
      currentArrow.setLatLng(nextPos);
      currentRange.setLatLng(nextPos);
      playerPosRef.current = nextPos;
      animRef.current.displayPos = nextPos;
    };

    const getDisplayPos = () => {
      const current = animRef.current.displayPos || [marker.getLatLng().lat, marker.getLatLng().lng];
      if (!Number.isFinite(current[0]) || !Number.isFinite(current[1])) {
        return render;
      }
      return current;
    };

    const ensureFollowAnimation = () => {
      if (animRef.current.raf) return;
      const tick = (timestamp) => {
        const currentMarker = overlayRef.current.player;
        if (!currentMarker) {
          animRef.current.raf = null;
          animRef.current.lastFrameTs = 0;
          return;
        }

        let desired =
          animRef.current.desiredPos ||
          animRef.current.displayPos ||
          [currentMarker.getLatLng().lat, currentMarker.getLatLng().lng];
        const motion = animRef.current.motion;
        if (
          motion &&
          Number.isFinite(motion.anchorPos?.[0]) &&
          Number.isFinite(motion.anchorPos?.[1]) &&
          Number.isFinite(motion.headingDeg) &&
          Number.isFinite(motion.speedMps) &&
          motion.speedMps > HEADING_SPEED_THRESHOLD_MPS &&
          timestamp - Number(motion.updatedAtTs || 0) <= MOTION_GRACE_MS
        ) {
          const travelMeters = Math.max(0, ((timestamp - Number(motion.anchorTs || timestamp)) / 1000) * motion.speedMps);
          let projected = moveByBearing(motion.anchorPos, motion.headingDeg, travelMeters);
          if (
            Array.isArray(motion.targetPos) &&
            Number.isFinite(motion.targetPos[0]) &&
            Number.isFinite(motion.targetPos[1])
          ) {
            const totalMeters = haversineMeters(motion.anchorPos, motion.targetPos);
            if (travelMeters >= totalMeters || haversineMeters(projected, motion.targetPos) <= SNAP_DISTANCE_METERS) {
              projected = motion.targetPos;
              animRef.current.motion = null;
            }
          }
          desired = projected;
        } else if (motion && timestamp - Number(motion.updatedAtTs || 0) > MOTION_GRACE_MS) {
          animRef.current.motion = null;
        }
        const current = animRef.current.displayPos || [currentMarker.getLatLng().lat, currentMarker.getLatLng().lng];
        const desiredPos = [Number(desired[0]), Number(desired[1])];
        const currentPos = [Number(current[0]), Number(current[1])];
        if (
          !Number.isFinite(desiredPos[0]) ||
          !Number.isFinite(desiredPos[1]) ||
          !Number.isFinite(currentPos[0]) ||
          !Number.isFinite(currentPos[1])
        ) {
          animRef.current.raf = null;
          animRef.current.lastFrameTs = 0;
          return;
        }

        const distance = haversineMeters(currentPos, desiredPos);
        const motionActive = Boolean(animRef.current.motion);
        if (distance <= SNAP_DISTANCE_METERS && !motionActive) {
          setDisplayPos(desiredPos);
          animRef.current.raf = null;
          animRef.current.lastFrameTs = 0;
          return;
        }

        const prevTs = animRef.current.lastFrameTs || timestamp;
        const dt = Math.max(8, Math.min(64, timestamp - prevTs));
        animRef.current.lastFrameTs = timestamp;
        const alphaRaw = 1 - Math.exp(-dt / FOLLOW_EASE_MS);
        const alpha = Math.max(FOLLOW_MIN_ALPHA, Math.min(FOLLOW_MAX_ALPHA, alphaRaw));
        const nextPos = [
          currentPos[0] + (desiredPos[0] - currentPos[0]) * alpha,
          currentPos[1] + (desiredPos[1] - currentPos[1]) * alpha
        ];
        setDisplayPos(nextPos);
        animRef.current.raf = requestAnimationFrame(tick);
      };

      animRef.current.lastFrameTs = 0;
      animRef.current.raf = requestAnimationFrame(tick);
    };

    if (!animRef.current.displayPos) {
      setDisplayPos(render);
    }

    const baseDisplay = getDisplayPos();
    const serverJumpMeters = haversineMeters(baseDisplay, render);
    const shouldSnapToRender =
      !animRef.current.lastRender || serverJumpMeters > TELEPORT_SNAP_METERS;
    if (shouldSnapToRender) {
      if (animRef.current.raf) {
        cancelAnimationFrame(animRef.current.raf);
        animRef.current.raf = null;
      }
      animRef.current.motion = null;
      animRef.current.lastFrameTs = 0;
      setDisplayPos(render);
    }

    let motionHeading = NaN;
    if (hasPreRenderDelta) {
      motionHeading = bearingDeg(render, preRender);
    } else if (Number.isFinite(headingFromPayload)) {
      motionHeading = headingFromPayload;
    }
    if (hasMotionSignal && Number.isFinite(motionHeading)) {
      const fallbackSpeedMps = Math.max(0, haversineMeters(render, preRender) * 5);
      const resolvedSpeedMps = movingBySpeed ? speedMps : fallbackSpeedMps;
      if (resolvedSpeedMps > HEADING_SPEED_THRESHOLD_MPS) {
        animRef.current.motion = {
          anchorPos: preRender,
          anchorTs: performance.now(),
          updatedAtTs: performance.now(),
          headingDeg: motionHeading,
          speedMps: resolvedSpeedMps,
          targetPos: hasRouteTarget ? routeTarget : null
        };
      } else {
        animRef.current.motion = null;
      }
    } else {
      animRef.current.motion = null;
    }

    animRef.current.desiredPos = serverDesiredPos;
    const fromPos = getDisplayPos();
    const motionDistance = haversineMeters(fromPos, serverDesiredPos);
    if (motionDistance <= SNAP_DISTANCE_METERS && !animRef.current.motion) {
      setDisplayPos(serverDesiredPos);
      if (animRef.current.raf) {
        cancelAnimationFrame(animRef.current.raf);
        animRef.current.raf = null;
      }
      animRef.current.lastFrameTs = 0;
    } else {
      ensureFollowAnimation();
    }

    const prevRender = animRef.current.lastRender || render;
    const movedByRender = haversineMeters(prevRender, render) > SNAP_DISTANCE_METERS;
    const movedByDisplay = motionDistance > SNAP_DISTANCE_METERS;
    let nextHeading = NaN;
    if (movedByDisplay) {
      nextHeading = bearingDeg(fromPos, serverDesiredPos);
    }
    if (!Number.isFinite(nextHeading) && movedByRender) {
      nextHeading = bearingDeg(prevRender, render);
    }
    if (!Number.isFinite(nextHeading) && hasPreRenderDelta) {
      nextHeading = bearingDeg(render, preRender);
    }
    if (!Number.isFinite(nextHeading) && Number.isFinite(headingFromPayload) && movingBySpeed) {
      nextHeading = headingFromPayload;
    }
    if (Number.isFinite(nextHeading)) {
      const currentHeading = animRef.current.bearing || 0;
      const normalizedHeading = currentHeading + shortestDeltaDeg(currentHeading, nextHeading);
      animRef.current.bearing = normalizedHeading;
      applyBearingRef.current(normalizedHeading);
    }
    animRef.current.lastRender = render;

    if (overlayRef.current.preRenderPath) {
      overlayRef.current.preRenderPath.remove();
      overlayRef.current.preRenderPath = null;
    }
    if (hasPreRenderDelta) {
      overlayRef.current.preRenderPath = leafletRef.current
        ?.polyline([render, preRender], {
          color: '#7fc9ff',
          dashArray: '3 5',
          weight: 2,
          opacity: 0.75
        })
        .addTo(mapRef.current);
    }
  }, [mapReady, moveTarget?.lat, moveTarget?.latitude, moveTarget?.lng, moveTarget?.longitude, player?.headingDeg, player?.latitude, player?.longitude, player?.preRenderLatitude, player?.preRenderLongitude, player?.renderLatitude, player?.renderLongitude, player?.speedMps]);

  useEffect(() => {
    const map = mapRef.current;
    const L = leafletRef.current;
    const overlays = overlayRef.current;
    if (!mapReady || !map || !L || !overlays.player) return;

    const render = [
      Number(player?.renderLatitude ?? player?.latitude),
      Number(player?.renderLongitude ?? player?.longitude)
    ];
    const display = animRef.current.displayPos;
    const pathStart =
      Array.isArray(display) && Number.isFinite(display[0]) && Number.isFinite(display[1]) ? display : render;
    const hasRender = Number.isFinite(render[0]) && Number.isFinite(render[1]);
    const hasTarget =
      Array.isArray(pendingTarget) &&
      Number.isFinite(Number(pendingTarget[0])) &&
      Number.isFinite(Number(pendingTarget[1]));

    if (overlays.path) {
      overlays.path.remove();
      overlays.path = null;
    }
    if (overlays.target) {
      overlays.target.remove();
      overlays.target = null;
    }
    if (!hasRender || !hasTarget) return;

    const target = [Number(pendingTarget[0]), Number(pendingTarget[1])];
    overlays.path = L.polyline([pathStart, target], {
      color: '#f6c356',
      dashArray: '6 6',
      weight: 2,
      opacity: 0.9
    }).addTo(map);
    overlays.target = L.circleMarker(target, {
      radius: 5,
      color: '#f6c356',
      weight: 2,
      fillColor: '#f6c356',
      fillOpacity: 0.25,
      interactive: false
    }).addTo(map);

    if (haversineMeters(render, target) <= 4) {
      setPendingTarget(null);
    }
  }, [mapReady, pendingTarget, player?.latitude, player?.longitude, player?.renderLatitude, player?.renderLongitude]);

  useEffect(() => {
    const map = mapRef.current;
    const L = leafletRef.current;
    const overlays = overlayRef.current;
    if (!mapReady || !map || !L || !overlays.portals || !overlays.links || !overlays.fields || !overlays.nearby) {
      return;
    }

    overlays.fields.clearLayers();
    fields.forEach((field) => {
      const points = (field.points || []).map((pt) => [pt.lat, pt.lng]);
      const style = styleForFaction(field.faction);
      if (points.length >= 3) {
        L.polygon(points, {
          color: style.stroke,
          weight: 2,
          fillColor: style.fill,
          fillOpacity: 0.18
        }).addTo(overlays.fields);
      }
    });

    overlays.links.clearLayers();
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
      ).addTo(overlays.links);
    });

    overlays.portals.clearLayers();
    portals.forEach((portal) => {
      const factionClass = factionToClass(portal.faction);
      const levelClass = portalLevelClass(portal.level || 1);
      const ring = buildResonatorRing(portal.resonators || []);
      const modRings = buildModRings(portal.mods || []);
      const isSource = linkMode?.active && linkMode?.fromPortalId === portal.id;
      const icon = L.divIcon({
        className: 'portal-marker',
        html: `<div class="portal ${factionClass} ${levelClass} ${isSource ? 'link-source' : ''}"><span class="portal-level-text">L${portal.level || 1}</span>${ring}${modRings}</div>`,
        iconSize: [54, 54],
        iconAnchor: [27, 27]
      });
      const marker = L.marker([portal.lat, portal.lng], { icon }).addTo(overlays.portals);
      marker.on('click', () => onOpenPortal?.(portal));
    });

    overlays.nearby.clearLayers();
    nearbyPlayers.forEach((agent) => {
      const renderLat = Number(agent.renderLatitude ?? agent.latitude);
      const renderLon = Number(agent.renderLongitude ?? agent.longitude);
      if (!Number.isFinite(renderLat) || !Number.isFinite(renderLon)) return;
      L.circleMarker([renderLat, renderLon], {
        radius: 5,
        color: '#f6c356',
        weight: 1,
        fillColor: '#f6c356',
        fillOpacity: 0.7
      }).addTo(overlays.nearby);

      const preRenderLat = Number(agent.preRenderLatitude ?? renderLat);
      const preRenderLon = Number(agent.preRenderLongitude ?? renderLon);
      if (!Number.isFinite(preRenderLat) || !Number.isFinite(preRenderLon)) return;

      if (Math.abs(preRenderLat - renderLat) <= 1e-7 && Math.abs(preRenderLon - renderLon) <= 1e-7) {
        return;
      }

      L.polyline(
        [
          [renderLat, renderLon],
          [preRenderLat, preRenderLon]
        ],
        {
          color: '#7fc9ff',
          dashArray: '2 4',
          weight: 1,
          opacity: 0.6
        }
      ).addTo(overlays.nearby);

      L.circleMarker([preRenderLat, preRenderLon], {
        radius: 3,
        color: '#7fc9ff',
        weight: 1,
        fillColor: '#7fc9ff',
        fillOpacity: 0.35
      }).addTo(overlays.nearby);
    });
  }, [fields, links, linkMode, mapReady, nearbyPlayers, onOpenPortal, portals]);

  useEffect(() => {
    const L = leafletRef.current;
    const overlays = overlayRef.current;
    if (!mapReady || !L || !overlays.charge) return undefined;

    const render = () => {
      overlays.charge.clearLayers();
      if (!attackChargeFx?.active) return;

      const markerPos = overlays.player?.getLatLng();
      const latitude = Number(markerPos?.lat ?? player?.renderLatitude ?? player?.latitude);
      const longitude = Number(markerPos?.lng ?? player?.renderLongitude ?? player?.longitude);
      if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) return;

      const now = Date.now();
      const startAt = Number(attackChargeFx.startAt || now);
      const durationMs = Math.max(1, Number(attackChargeFx.durationMs || 1));
      const progress = Math.max(0, Math.min(1, (now - startAt) / durationMs));
      const radius = Math.max(1, ACTION_RADIUS_METERS * (1 - progress));
      const alpha = Math.max(0.2, 0.8 * (1 - progress));

      L.circle([latitude, longitude], {
        radius,
        color: '#ffce00',
        weight: 2,
        opacity: alpha,
        fill: false,
        className: 'attack-charge-ring',
        interactive: false
      }).addTo(overlays.charge);
    };

    render();
    if (!attackChargeFx?.active) return undefined;
    const timer = window.setInterval(render, 50);
    return () => {
      window.clearInterval(timer);
      overlays.charge.clearLayers();
    };
  }, [attackChargeFx, mapReady, player?.latitude, player?.longitude, player?.renderLatitude, player?.renderLongitude]);

  useEffect(() => {
    const L = leafletRef.current;
    const overlays = overlayRef.current;
    if (!mapReady || !L || !overlays.pulses) return undefined;

    const render = () => {
      const now = Date.now();
      overlays.pulses.clearLayers();
      (Array.isArray(attackPulses) ? attackPulses : []).forEach((pulse) => {
        const createdAt = Number(pulse?.createdAt || 0);
        const progress = (now - createdAt) / 900;
        if (progress < 0 || progress > 1) return;
        const latitude = Number(pulse?.latitude);
        const longitude = Number(pulse?.longitude);
        const maxRadius = Number(pulse?.radius || 0);
        if (!Number.isFinite(latitude) || !Number.isFinite(longitude) || !Number.isFinite(maxRadius) || maxRadius <= 0) {
          return;
        }
        const eased = Math.min(1, Math.max(0, progress));
        const radius = maxRadius * eased;
        const alpha = Math.max(0, 0.7 * (1 - eased));
        L.circle([latitude, longitude], {
          radius,
          color: '#ffce00',
          weight: 3 - eased * 1.5,
          opacity: alpha,
          fill: false,
          className: 'attack-pulse-ring',
          interactive: false
        }).addTo(overlays.pulses);
      });
    };

    render();
    const timer = window.setInterval(render, 50);
    return () => {
      window.clearInterval(timer);
      overlays.pulses.clearLayers();
    };
  }, [attackPulses, mapReady]);

  useEffect(() => {
    const handleKeyDown = (event) => {
      if ((event.key === 'h' || event.key === 'H') && mapRef.current) {
        mapRef.current.setView(playerPosRef.current, mapRef.current.getZoom());
      }
      if ((event.key === 'Escape' || event.key === 'Esc') && linkMode?.active) {
        onCancelLinkMode?.();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [linkMode?.active, onCancelLinkMode]);

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
      {linkMode?.active ? (
        <div className="map-link-hint">
          <span>Link 模式：点击目标 Portal</span>
          <button className="map-link-hint-close" onClick={() => onCancelLinkMode?.()}>
            退出
          </button>
        </div>
      ) : null}
    </section>
  );
}

function portalLevelClass(level) {
  if (level <= 3) return 'portal-lvl-low';
  if (level <= 6) return 'portal-lvl-mid';
  return 'portal-lvl-high';
}

function buildResonatorRing(resonators = []) {
  const posByIndex = ['nw', 'w', 'sw', 's', 'se', 'e', 'ne', 'n'];
  const slots = Array.from({ length: 8 }, (_, idx) => {
    const hasRes = Boolean(resonators[idx]?.level);
    if (!hasRes) return '';
    return `<span class="portal-res-slot on pos-${posByIndex[idx] || idx}"></span>`;
  }).join('');
  return `<span class="portal-reso-ring">${slots}</span>`;
}

function buildModRings(mods = []) {
  const rings = mods
    .map((mod, idx) => {
      if (!mod?.subtype) return '';
      const rarity = mod.rarity || 'C';
      return `<span class="portal-mod-ring slot-${idx} rarity-${rarity}"></span>`;
    })
    .join('');
  return `<span class="portal-mod-rings">${rings}</span>`;
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

function moveByBearing(from, bearing, distanceMeters) {
  const toRad = (d) => (d * Math.PI) / 180;
  const toDeg = (r) => (r * 180) / Math.PI;
  const earthRadius = 6371000;
  const angular = distanceMeters / earthRadius;
  const lat1 = toRad(from[0]);
  const lon1 = toRad(from[1]);
  const heading = toRad(bearing);

  const sinLat2 =
    Math.sin(lat1) * Math.cos(angular) +
    Math.cos(lat1) * Math.sin(angular) * Math.cos(heading);
  const lat2 = Math.asin(Math.max(-1, Math.min(1, sinLat2)));
  const lon2 =
    lon1 +
    Math.atan2(
      Math.sin(heading) * Math.sin(angular) * Math.cos(lat1),
      Math.cos(angular) - Math.sin(lat1) * Math.sin(lat2)
    );

  return [toDeg(lat2), toDeg(lon2)];
}

function shortestDeltaDeg(current, target) {
  return ((target - current + 540) % 360) - 180;
}
