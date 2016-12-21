package constant

const (
	MEDIA_SERVER_NAME     = "Digital Operating Room Media Server"
	PROXY_SERVER_NAME     = "Digital Operating Room Proxy Server"
	RECORD_SERVER_NAME    = "Digital Operating Room Record Server"
	MEDIA_SERVER_VERSION  = "1.0.0.3"
	PROXY_SERVER_VERSION  = "1.0.0.3"
	RECORD_SERVER_VERSION = "1.0.0.3"
	START_MEDIA_SERVER    = "Start " + MEDIA_SERVER_NAME + "."
	START_PROXY_SERVER    = "Start " + PROXY_SERVER_NAME + "."
	START_RECORD_SERVER   = "Start " + RECORD_SERVER_NAME + "."
	CLOSE_MEDIA_SERVER    = "Closed media server."
	CLOSE_PROXY_SERVER    = "Closed proxy server."
	CLOSE_RECORD_SERVER   = "Closed record server."

	START_AS_DAEMON = "Start as a daemon."

	SUCCESS_READ_CONFIG   = "Success for reading config information."
	SUCCESS_READ_DATABASE = "Success for reading database information."

	FAILED_READ_CONFIG   = "Failed to read configure file."
	FAILED_CREATE_SERVER = "Failed to create RTSP server."

	DORMS_CONFIG_FILE = "dorms.conf"
	DORPS_CONFIG_FILE = "dorps.conf"
	DORRS_CONFIG_FILE = "dorrs.conf"

	HELP_MESSAGE = "/h /help\tPrint this message and quit.\n/v /version\tPrint the version of the server and quit.\n" +
		"/i /install\n/u /uninstall"
	HELP_DAEMON = "/d /daemon\tStart Server as a daemon"
)
