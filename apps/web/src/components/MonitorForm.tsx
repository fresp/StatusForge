import React, { useState } from 'react';

interface Component {
  id: string;
  name: string;
}

interface SubComponent {
  id: string;
  name: string;
  componentId: string;
}

interface MonitorFormProps {
  components: Component[];
  subcomponents: SubComponent[];
  onSave: (monitor: any) => void;
  onCancel: () => void;
}

interface FormState {
  name: string;
  type: string;
  target: string;
  sslThresholds: string;
  interval: number;
  timeout: number;
  componentId: string;
  subcomponentId: string;
}

const MONITOR_TYPES = ['http', 'tcp', 'dns', 'ping', 'ssl'];

export function MonitorForm({ components, subcomponents, onSave, onCancel }: MonitorFormProps) {
  const [form, setForm] = useState<FormState>({
    name: '',
    type: 'http',
    target: '',
    sslThresholds: '30,14,7',
    interval: 60,
    timeout: 30,
    componentId: '',
    subcomponentId: '',
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setForm(prev => ({
      ...prev,
      [name]: name === 'interval' || name === 'timeout' ? parseInt(value) : value
    }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    // Validate that either component or subcomponent is selected but not both
    if ((!form.componentId && !form.subcomponentId) || (form.componentId && form.subcomponentId)) {
      alert('Please select either a parent component or a subcomponent, but not both');
      return;
    }

    const monitor = {
      name: form.name,
      type: form.type,
      target: form.target,
      ...(form.type === 'ssl'
        ? {
            sslThresholds: form.sslThresholds
              .split(',')
              .map(v => parseInt(v.trim(), 10))
              .filter(v => Number.isFinite(v) && v > 0),
          }
        : {}),
      intervalSeconds: form.interval,
      timeoutSeconds: form.timeout,
      ...(form.componentId ? { componentId: form.componentId } : {}),
      ...(form.subcomponentId ? { subComponentId: form.subcomponentId } : {}),
    };

    onSave(monitor);
  };

  const getSubcomponentsForComponent = (componentId: string) => {
    return subcomponents.filter(sc => sc.componentId === componentId);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
        <input
          type="text"
          name="name"
          value={form.name}
          onChange={handleChange}
          required
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
        <select
          name="type"
          value={form.type}
          onChange={handleChange}
          required
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {MONITOR_TYPES.map(type => (
            <option key={type} value={type}>{type.toUpperCase()}</option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Target</label>
        <input
          type="text"
          name="target"
          value={form.target}
          onChange={handleChange}
          required
          placeholder="e.g., https://example.com, 192.168.1.1:8080"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {form.type === 'ssl' && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">SSL Alert Thresholds (days)</label>
          <input
            type="text"
            name="sslThresholds"
            value={form.sslThresholds}
            onChange={handleChange}
            placeholder="30,14,7"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      )}

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Check Interval (seconds)</label>
          <input
            type="number"
            name="interval"
            value={form.interval}
            onChange={handleChange}
            min="10"
            required
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Timeout (seconds)</label>
          <input
            type="number"
            name="timeout"
            value={form.timeout}
            onChange={handleChange}
            min="1"
            required
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      {/* Component/Subcomponent association */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Parent Component (optional)</label>
          <select
            name="componentId"
            value={form.componentId}
            onChange={e => {
              const value = e.target.value;
              setForm(prev => ({
                ...prev,
                componentId: value,
                subcomponentId: '' // Reset subcomponent if component is selected
              }));
            }}
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">Select component</option>
            {components.map(comp => (
              <option key={comp.id} value={comp.id}>{comp.name}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            {form.componentId ? 'Subcomponent within selected component' : 'Or Select Subcomponent directly (optional)'}
          </label>
          <select
            name="subcomponentId"
            value={form.subcomponentId}
            onChange={e => {
              const value = e.target.value;
              setForm(prev => ({
                ...prev,
                subcomponentId: value,
                componentId: ''  // Reset component if subcomponent is selected directly
              }));
            }}
            disabled={!form.componentId && !subcomponents.length}
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
          >
            <option value="">Select subcomponent</option>
            {(form.componentId ? getSubcomponentsForComponent(form.componentId) : subcomponents).map(sub => (
              <option key={sub.id} value={sub.id}>{sub.name}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="flex gap-3 pt-4">
        <button
          type="button"
          onClick={onCancel}
          className="flex-1 border border-gray-300 text-gray-700 rounded-lg py-2 text-sm hover:bg-gray-50"
        >
          Cancel
        </button>
        <button
          type="submit"
          className="flex-1 bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 text-sm font-medium"
        >
          Save Monitor
        </button>
      </div>
    </form>
  );
}
