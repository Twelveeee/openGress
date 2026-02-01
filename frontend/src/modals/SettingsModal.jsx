import React from 'react';
import Modal from '../components/Modal.jsx';

export default function SettingsModal({ open, onClose }) {
  return (
    <Modal open={open} onClose={onClose}>
      <header className="modal-header">
        <h3>设置</h3>
      </header>
      <div className="settings-body">
        <label>
          <input type="checkbox" /> 自动 Hack
        </label>
        <label>
          <input type="checkbox" /> 显示连接线
        </label>
        <label>
          <input type="checkbox" /> 显示控制场
        </label>
      </div>
    </Modal>
  );
}
