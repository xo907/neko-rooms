// Every option of the neko server, generated from neko's own option list
// (webpage/docs/configuration/help.json at the neko version rooms run).
// Options neko-rooms sets for every room are managed: shown, not editable.

export interface NekoOption {
  env: string
  group: string
  type: string // boolean, int, float, string, strings, duration, array, object
  default: string
  description: string
  managed: boolean
}

export const NEKO_OPTIONS: NekoOption[] = [
  {
    "env": "NEKO_CAPTURE_AUDIO_CODEC",
    "group": "Capture",
    "type": "string",
    "default": "opus",
    "description": "audio codec to be used",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_AUDIO_DEVICE",
    "group": "Capture",
    "type": "string",
    "default": "audio_output.monitor",
    "description": "pulseaudio device to capture",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_AUDIO_PIPELINE",
    "group": "Capture",
    "type": "string",
    "default": "",
    "description": "gstreamer pipeline used for audio streaming",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_BROADCAST_AUDIO_BITRATE",
    "group": "Capture",
    "type": "int",
    "default": "128",
    "description": "broadcast audio bitrate in KB/s",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_BROADCAST_AUTOSTART",
    "group": "Capture",
    "type": "boolean",
    "default": "true",
    "description": "automatically start broadcasting when neko starts and broadcast_url is set",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_BROADCAST_PIPELINE",
    "group": "Capture",
    "type": "string",
    "default": "",
    "description": "gstreamer pipeline used for broadcasting",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_BROADCAST_PRESET",
    "group": "Capture",
    "type": "string",
    "default": "veryfast",
    "description": "broadcast speed preset for h264 encoding",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_BROADCAST_URL",
    "group": "Capture",
    "type": "string",
    "default": "",
    "description": "initial URL for broadcasting, setting this value will automatically start broadcasting",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_BROADCAST_VIDEO_BITRATE",
    "group": "Capture",
    "type": "int",
    "default": "4096",
    "description": "broadcast video bitrate in KB/s",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_MICROPHONE_DEVICE",
    "group": "Capture",
    "type": "string",
    "default": "audio_input",
    "description": "pulseaudio device used for microphone",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_MICROPHONE_ENABLED",
    "group": "Capture",
    "type": "boolean",
    "default": "true",
    "description": "enable microphone stream",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_SCREENCAST_ENABLED",
    "group": "Capture",
    "type": "boolean",
    "default": "false",
    "description": "enable screencast",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_SCREENCAST_PIPELINE",
    "group": "Capture",
    "type": "string",
    "default": "",
    "description": "gstreamer pipeline used for screencasting",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_SCREENCAST_QUALITY",
    "group": "Capture",
    "type": "string",
    "default": "60",
    "description": "screencast JPEG quality",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_SCREENCAST_RATE",
    "group": "Capture",
    "type": "string",
    "default": "10/1",
    "description": "screencast frame rate",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_VIDEO_CODEC",
    "group": "Capture",
    "type": "string",
    "default": "vp8",
    "description": "video codec to be used",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_VIDEO_DISPLAY",
    "group": "Capture",
    "type": "string",
    "default": "",
    "description": "X display to capture",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_VIDEO_IDS",
    "group": "Capture",
    "type": "strings",
    "default": "",
    "description": "ordered list of video ids",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_VIDEO_PIPELINE",
    "group": "Capture",
    "type": "string",
    "default": "",
    "description": "shortcut for configuring only a single gstreamer pipeline, ignored if pipelines is set",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_VIDEO_PIPELINES",
    "group": "Capture",
    "type": "object",
    "default": "{}",
    "description": "pipelines config used for video streaming",
    "managed": true
  },
  {
    "env": "NEKO_CAPTURE_VIDEO_SHOW_POINTER",
    "group": "Capture",
    "type": "boolean",
    "default": "true",
    "description": "show mouse pointer in captured video, overrides show_pointer of all video pipelines",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_WEBCAM_DEVICE",
    "group": "Capture",
    "type": "string",
    "default": "/dev/video0",
    "description": "v4l2sink device used for webcam",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_WEBCAM_ENABLED",
    "group": "Capture",
    "type": "boolean",
    "default": "false",
    "description": "enable webcam stream",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_WEBCAM_HEIGHT",
    "group": "Capture",
    "type": "int",
    "default": "720",
    "description": "webcam stream height",
    "managed": false
  },
  {
    "env": "NEKO_CAPTURE_WEBCAM_WIDTH",
    "group": "Capture",
    "type": "int",
    "default": "1280",
    "description": "webcam stream width",
    "managed": false
  },
  {
    "env": "NEKO_DESKTOP_DISPLAY",
    "group": "Desktop",
    "type": "string",
    "default": "",
    "description": "X display to use for desktop sharing",
    "managed": true
  },
  {
    "env": "NEKO_DESKTOP_FILE_CHOOSER_DIALOG",
    "group": "Desktop",
    "type": "boolean",
    "default": "false",
    "description": "whether to handle file chooser dialog externally",
    "managed": false
  },
  {
    "env": "NEKO_DESKTOP_INPUT_ENABLED",
    "group": "Desktop",
    "type": "boolean",
    "default": "true",
    "description": "whether custom xf86 input driver should be used to handle touchscreen",
    "managed": false
  },
  {
    "env": "NEKO_DESKTOP_INPUT_SOCKET",
    "group": "Desktop",
    "type": "string",
    "default": "/tmp/xf86-input-neko.sock",
    "description": "socket path for custom xf86 input driver connection",
    "managed": true
  },
  {
    "env": "NEKO_DESKTOP_SCREEN",
    "group": "Desktop",
    "type": "string",
    "default": "1280x720@30",
    "description": "default screen size and framerate",
    "managed": true
  },
  {
    "env": "NEKO_DESKTOP_UNMINIMIZE",
    "group": "Desktop",
    "type": "boolean",
    "default": "true",
    "description": "automatically unminimize window when it is minimized",
    "managed": false
  },
  {
    "env": "NEKO_DESKTOP_UPLOAD_DROP",
    "group": "Desktop",
    "type": "boolean",
    "default": "true",
    "description": "whether drop upload is enabled",
    "managed": false
  },
  {
    "env": "NEKO_MEMBER_FILE_HASH",
    "group": "Members",
    "type": "boolean",
    "default": "true",
    "description": "member file provider: whether the passwords are hashed using sha256 or not (recommended)",
    "managed": true
  },
  {
    "env": "NEKO_MEMBER_FILE_PATH",
    "group": "Members",
    "type": "string",
    "default": "",
    "description": "member file provider: path to the file containing the users and their passwords",
    "managed": true
  },
  {
    "env": "NEKO_MEMBER_MULTIUSER_ADMIN_PASSWORD",
    "group": "Members",
    "type": "string",
    "default": "admin",
    "description": "member multiuser provider: password for admin users",
    "managed": true
  },
  {
    "env": "NEKO_MEMBER_MULTIUSER_ADMIN_PROFILE",
    "group": "Members",
    "type": "object",
    "default": "{}",
    "description": "member multiuser provider: profile template for admin users",
    "managed": true
  },
  {
    "env": "NEKO_MEMBER_MULTIUSER_USER_PASSWORD",
    "group": "Members",
    "type": "string",
    "default": "neko",
    "description": "member multiuser provider: password for regular users",
    "managed": true
  },
  {
    "env": "NEKO_MEMBER_MULTIUSER_USER_PROFILE",
    "group": "Members",
    "type": "object",
    "default": "{}",
    "description": "member multiuser provider: profile template for regular users",
    "managed": true
  },
  {
    "env": "NEKO_MEMBER_OBJECT_USERS",
    "group": "Members",
    "type": "array",
    "default": "[]",
    "description": "member object provider: list of users with their passwords and profiles",
    "managed": true
  },
  {
    "env": "NEKO_MEMBER_PROVIDER",
    "group": "Members",
    "type": "string",
    "default": "multiuser",
    "description": "selected member provider",
    "managed": true
  },
  {
    "env": "NEKO_PLUGINS_DIR",
    "group": "Plugins",
    "type": "string",
    "default": "./bin/plugins",
    "description": "path to neko plugins to load",
    "managed": true
  },
  {
    "env": "NEKO_PLUGINS_ENABLED",
    "group": "Plugins",
    "type": "boolean",
    "default": "false",
    "description": "load plugins in runtime",
    "managed": true
  },
  {
    "env": "NEKO_PLUGINS_REQUIRED",
    "group": "Plugins",
    "type": "boolean",
    "default": "false",
    "description": "if true, neko will exit if there is an error when loading a plugin",
    "managed": true
  },
  {
    "env": "NEKO_SERVER_BIND",
    "group": "Server",
    "type": "string",
    "default": "127.0.0.1:8080",
    "description": "address/port/socket to serve neko",
    "managed": true
  },
  {
    "env": "NEKO_SERVER_CERT",
    "group": "Server",
    "type": "string",
    "default": "",
    "description": "path to the SSL cert used to secure the neko server",
    "managed": true
  },
  {
    "env": "NEKO_SERVER_CORS",
    "group": "Server",
    "type": "strings",
    "default": "",
    "description": "list of allowed origins for CORS, if empty CORS is disabled, if '*' is present all origins are allowed",
    "managed": false
  },
  {
    "env": "NEKO_SERVER_KEY",
    "group": "Server",
    "type": "string",
    "default": "",
    "description": "path to the SSL key used to secure the neko server",
    "managed": true
  },
  {
    "env": "NEKO_SERVER_METRICS",
    "group": "Server",
    "type": "boolean",
    "default": "true",
    "description": "enable prometheus metrics available at /metrics",
    "managed": false
  },
  {
    "env": "NEKO_SERVER_PATH_PREFIX",
    "group": "Server",
    "type": "string",
    "default": "/",
    "description": "path prefix for HTTP requests",
    "managed": true
  },
  {
    "env": "NEKO_SERVER_PPROF",
    "group": "Server",
    "type": "boolean",
    "default": "false",
    "description": "enable pprof endpoint available at /debug/pprof",
    "managed": false
  },
  {
    "env": "NEKO_SERVER_PROXY",
    "group": "Server",
    "type": "boolean",
    "default": "false",
    "description": "trust reverse proxy headers",
    "managed": true
  },
  {
    "env": "NEKO_SERVER_STATIC",
    "group": "Server",
    "type": "string",
    "default": "",
    "description": "path to neko client files to serve",
    "managed": true
  },
  {
    "env": "NEKO_SESSION_API_TOKEN",
    "group": "Session",
    "type": "string",
    "default": "",
    "description": "API token for interacting with external services",
    "managed": true
  },
  {
    "env": "NEKO_SESSION_CONTROL_PROTECTION",
    "group": "Session",
    "type": "boolean",
    "default": "false",
    "description": "users can gain control only if at least one admin is in the room",
    "managed": true
  },
  {
    "env": "NEKO_SESSION_COOKIE_DOMAIN",
    "group": "Session",
    "type": "string",
    "default": "",
    "description": "domain of the cookie",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_COOKIE_ENABLED",
    "group": "Session",
    "type": "boolean",
    "default": "false",
    "description": "whether cookies authentication should be enabled",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_COOKIE_EXPIRATION",
    "group": "Session",
    "type": "duration",
    "default": "24h0m0s",
    "description": "expiration of the cookie",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_COOKIE_HTTP_ONLY",
    "group": "Session",
    "type": "boolean",
    "default": "true",
    "description": "use http only cookies",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_COOKIE_NAME",
    "group": "Session",
    "type": "string",
    "default": "NEKO_SESSION",
    "description": "name of the cookie that holds token",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_COOKIE_PATH",
    "group": "Session",
    "type": "string",
    "default": "",
    "description": "path of the cookie",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_COOKIE_SECURE",
    "group": "Session",
    "type": "boolean",
    "default": "true",
    "description": "use secure cookies",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_FILE",
    "group": "Session",
    "type": "string",
    "default": "",
    "description": "if sessions should be stored in a file, otherwise they will be stored only in memory",
    "managed": true
  },
  {
    "env": "NEKO_SESSION_HEARTBEAT_INTERVAL",
    "group": "Session",
    "type": "int",
    "default": "10",
    "description": "interval in seconds for sending heartbeat messages",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_IMPLICIT_HOSTING",
    "group": "Session",
    "type": "boolean",
    "default": "false",
    "description": "allow implicit control switching",
    "managed": true
  },
  {
    "env": "NEKO_SESSION_INACTIVE_CURSORS",
    "group": "Session",
    "type": "boolean",
    "default": "false",
    "description": "show inactive cursors on the screen",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_LOCKED_CONTROLS",
    "group": "Session",
    "type": "boolean",
    "default": "false",
    "description": "whether controls should be locked for users initially",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_LOCKED_LOGINS",
    "group": "Session",
    "type": "boolean",
    "default": "false",
    "description": "whether logins should be locked for users initially",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_MERCIFUL_RECONNECT",
    "group": "Session",
    "type": "boolean",
    "default": "true",
    "description": "allow reconnecting to websocket even if previous connection was not closed",
    "managed": false
  },
  {
    "env": "NEKO_SESSION_PRIVATE_MODE",
    "group": "Session",
    "type": "boolean",
    "default": "false",
    "description": "whether private mode should be enabled initially",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_EPR",
    "group": "WebRTC",
    "type": "string",
    "default": "",
    "description": "limits the pool of ephemeral ports that ICE UDP connections can allocate from",
    "managed": true
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_DEBUG",
    "group": "WebRTC",
    "type": "boolean",
    "default": "false",
    "description": "enables debug logging for the bandwidth estimator",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_DIFF_THRESHOLD",
    "group": "WebRTC",
    "type": "float",
    "default": "0.15",
    "description": "how bigger the difference between estimated and stream bitrate must be to trigger upgrade/downgrade",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_DOWNGRADE_BACKOFF",
    "group": "WebRTC",
    "type": "duration",
    "default": "10s",
    "description": "how long to wait before downgrading again after previous downgrade",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_ENABLED",
    "group": "WebRTC",
    "type": "boolean",
    "default": "false",
    "description": "enables the bandwidth estimator",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_INITIAL_BITRATE",
    "group": "WebRTC",
    "type": "int",
    "default": "1000000",
    "description": "initial bitrate for the bandwidth estimator",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_PASSIVE",
    "group": "WebRTC",
    "type": "boolean",
    "default": "false",
    "description": "passive estimator mode, when it does not switch pipelines, only estimates",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_READ_INTERVAL",
    "group": "WebRTC",
    "type": "duration",
    "default": "2s",
    "description": "how often to read and process bandwidth estimation reports",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_STABLE_DURATION",
    "group": "WebRTC",
    "type": "duration",
    "default": "12s",
    "description": "how long to wait for stable connection (upward or neutral trend) before upgrading",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_STALLED_DURATION",
    "group": "WebRTC",
    "type": "duration",
    "default": "24s",
    "description": "how long to wait for stalled bandwidth estimation before downgrading",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_UNSTABLE_DURATION",
    "group": "WebRTC",
    "type": "duration",
    "default": "6s",
    "description": "how long to wait for stalled connection (neutral trend with low bandwidth) before downgrading",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ESTIMATOR_UPGRADE_BACKOFF",
    "group": "WebRTC",
    "type": "duration",
    "default": "5s",
    "description": "how long to wait before upgrading again after previous upgrade",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ICELITE",
    "group": "WebRTC",
    "type": "boolean",
    "default": "false",
    "description": "configures whether or not the ICE agent should be a lite agent",
    "managed": true
  },
  {
    "env": "NEKO_WEBRTC_ICESERVERS_BACKEND",
    "group": "WebRTC",
    "type": "array",
    "default": "[]",
    "description": "STUN and TURN servers used by the backend",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ICESERVERS_FRONTEND",
    "group": "WebRTC",
    "type": "array",
    "default": "[]",
    "description": "STUN and TURN servers used by the frontend",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_ICETRICKLE",
    "group": "WebRTC",
    "type": "boolean",
    "default": "true",
    "description": "configures whether cadidates should be sent asynchronously using Trickle ICE",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_IP_RETRIEVAL_URL",
    "group": "WebRTC",
    "type": "string",
    "default": "https://checkip.amazonaws.com",
    "description": "URL address used for retrieval of the external IP address",
    "managed": false
  },
  {
    "env": "NEKO_WEBRTC_NAT1TO1",
    "group": "WebRTC",
    "type": "strings",
    "default": "",
    "description": "sets a list of external IP addresses of 1:1 (D)NAT and a candidate type for which the external IP address is used",
    "managed": true
  },
  {
    "env": "NEKO_WEBRTC_TCPMUX",
    "group": "WebRTC",
    "type": "int",
    "default": "",
    "description": "single TCP mux port for all peers",
    "managed": true
  },
  {
    "env": "NEKO_WEBRTC_UDPMUX",
    "group": "WebRTC",
    "type": "int",
    "default": "",
    "description": "single UDP mux port for all peers, replaces EPR",
    "managed": true
  },
  {
    "env": "NEKO_CONFIG",
    "group": "General",
    "type": "string",
    "default": "",
    "description": "configuration file path",
    "managed": true
  },
  {
    "env": "NEKO_DEBUG",
    "group": "General",
    "type": "boolean",
    "default": "false",
    "description": "enable debug mode",
    "managed": false
  },
  {
    "env": "NEKO_LOG_DIR",
    "group": "General",
    "type": "string",
    "default": "",
    "description": "logging directory to store logs",
    "managed": false
  },
  {
    "env": "NEKO_LOG_JSON",
    "group": "General",
    "type": "boolean",
    "default": "false",
    "description": "logs in JSON format",
    "managed": false
  },
  {
    "env": "NEKO_LOG_LEVEL",
    "group": "General",
    "type": "string",
    "default": "info",
    "description": "set log level (trace, debug, info, warn, error, fatal, panic, disabled)",
    "managed": false
  },
  {
    "env": "NEKO_LOG_NOCOLOR",
    "group": "General",
    "type": "boolean",
    "default": "false",
    "description": "no ANSI colors in non-JSON output",
    "managed": false
  },
  {
    "env": "NEKO_LOG_TIME",
    "group": "General",
    "type": "string",
    "default": "unix",
    "description": "time format used in logs (unix, unixms, unixmicro)",
    "managed": false
  }
]
