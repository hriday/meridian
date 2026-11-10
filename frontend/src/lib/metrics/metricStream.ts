import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";

// subscribe wires a monitor's Wails event stream ("metrics:<id>") to a callback
// and returns an unsubscribe. The caller generates the id and subscribes BEFORE
// starting the backend monitor, so no snapshot is missed.
export function subscribe(streamID: string, onSnapshot: (s: any) => void): () => void {
  const name = "metrics:" + streamID;
  EventsOn(name, (s: any) => onSnapshot(s));
  return () => EventsOff(name);
}

// A short random stream id.
export function newStreamID(): string {
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}
