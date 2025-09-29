# Time Tracking

Simple time tracking for user sessions with automatic duplicate prevention and cleanup.

## Usage

```tsx
import { useTimeTracking } from './timeTracking';

export default function MyComponent() {
  const { startTracking } = useTimeTracking();

  useEffect(() => {
    startTracking(); // Starts tracking for current route
  }, [startTracking]);

  return <div>My Component</div>;
}
```

## Features

- **Automatic route detection** - Extracts story ID from `/stories/123` URLs
- **Duplicate prevention** - Multiple `startTracking()` calls return same session
- **Auto cleanup** - Handles component unmount and page leave events
- **Beacon API** - Reliable tracking on page unload/visibility change
- **Global state** - Shared across components to prevent conflicts

## API

- `startTracking(route?: string)` - Start tracking (optional custom route)
- `endTracking(trackingId?: number)` - Manual end (usually not needed)

Backend handles:
- Sessions > 5 minutes auto-close and create new ones
- Duplicate end requests are ignored
- Same user/route/story returns existing session ID