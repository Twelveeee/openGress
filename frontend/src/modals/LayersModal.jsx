import React from 'react';
import Modal from '../components/Modal.jsx';

export default function LayersModal({ open, onClose }) {
  return (
    <Modal open={open} onClose={onClose}>
      <header className="modal-header">
        <h3>图层切换</h3>
      </header>
      <div className="settings-body">
        <label>
          <input type="radio" name="layer" /> OSM
        </label>
        <label>
          <input type="radio" name="layer" /> AMap
        </label>
        <label>
          <input type="radio" name="layer" /> Google Maps
        </label>
      </div>
    </Modal>
  );
}
