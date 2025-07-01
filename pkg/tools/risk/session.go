package risk

import "time"

type TradingSession int

const (
	SessionClosed TradingSession = iota
	SessionSydney
	SessionTokyo
	SessionLondon
	SessionNewYork
	SessionSydneyTokyo   // Overlap
	SessionTokyoLondon   // Overlap
	SessionLondonNewYork // Overlap
)

type FxSessionInfo struct {
	Session     TradingSession
	Description string
}

type sessionSchedule struct {
	Name      string
	OpenHour  int // Hour in session's local time
	OpenMin   int
	CloseHour int
	CloseMin  int
	Timezone  string
}

var sessionSchedules = map[TradingSession]sessionSchedule{
	SessionSydney: {
		Name:      "Sydney",
		OpenHour:  22, // 10 PM previous day
		OpenMin:   0,
		CloseHour: 7, // 7 AM
		CloseMin:  0,
		Timezone:  "Australia/Sydney",
	},
	SessionTokyo: {
		Name:      "Tokyo",
		OpenHour:  0, // Midnight
		OpenMin:   0,
		CloseHour: 9, // 9 AM
		CloseMin:  0,
		Timezone:  "Asia/Tokyo",
	},
	SessionLondon: {
		Name:      "London",
		OpenHour:  8, // 8 AM
		OpenMin:   0,
		CloseHour: 17, // 5 PM
		CloseMin:  0,
		Timezone:  "Europe/London",
	},
	SessionNewYork: {
		Name:      "New York",
		OpenHour:  8, // 8 AM
		OpenMin:   0,
		CloseHour: 17, // 5 PM
		CloseMin:  0,
		Timezone:  "America/New_York",
	},
}

func GetCurrentSession() FxSessionInfo {
	now := time.Now()
	return GetSessionAtTime(now)
}

func GetSessionAtTime(t time.Time) FxSessionInfo {

	sydneyActive := isSessionActive(t, SessionSydney)
	tokyoActive := isSessionActive(t, SessionTokyo)
	londonActive := isSessionActive(t, SessionLondon)
	newYorkActive := isSessionActive(t, SessionNewYork)

	// Determine overlaps and primary session
	switch {
	case londonActive && newYorkActive:
		return FxSessionInfo{
			Session:     SessionLondonNewYork,
			Description: "London-New York Overlap",
		}
	case tokyoActive && londonActive:
		return FxSessionInfo{
			Session:     SessionTokyoLondon,
			Description: "Tokyo-London Overlap",
		}
	case sydneyActive && tokyoActive:
		return FxSessionInfo{
			Session:     SessionSydneyTokyo,
			Description: "Sydney-Tokyo Overlap",
		}
	case newYorkActive:
		return FxSessionInfo{
			Session:     SessionNewYork,
			Description: "New York",
		}
	case londonActive:
		return FxSessionInfo{
			Session:     SessionLondon,
			Description: "London",
		}
	case tokyoActive:
		return FxSessionInfo{
			Session:     SessionTokyo,
			Description: "Tokyo",
		}
	case sydneyActive:
		return FxSessionInfo{
			Session:     SessionSydney,
			Description: "Sydney",
		}
	default:
		return FxSessionInfo{
			Session:     SessionClosed,
			Description: "Market Closed",
		}
	}
}

func isSessionActive(t time.Time, session TradingSession) bool {
	schedule, exists := sessionSchedules[session]
	if !exists {
		return false
	}

	// Load the session's timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		// Fallback to UTC if timezone loading fails
		loc = time.UTC
	}

	// Convert current time to session's local time
	localTime := t.In(loc)

	// Get current time components
	currentHour := localTime.Hour()
	currentMin := localTime.Minute()

	// Forex market is closed from Friday 5 PM EST to Sunday 5 PM EST
	// Check if it's weekend (in EST)
	estLoc, _ := time.LoadLocation("America/New_York")
	estTime := t.In(estLoc)
	estWeekday := estTime.Weekday()
	estHour := estTime.Hour()

	// Market closed from Friday 5 PM EST to Sunday 5 PM EST
	if estWeekday == time.Friday && estHour >= 17 {
		return false
	}
	if estWeekday == time.Saturday {
		return false
	}
	if estWeekday == time.Sunday && estHour < 17 {
		return false
	}

	// Convert current time to minutes since midnight
	currentMinutes := currentHour*60 + currentMin

	// Handle sessions that cross midnight
	if schedule.CloseHour < schedule.OpenHour {
		// Session spans midnight (like Sydney)
		openMinutes := schedule.OpenHour*60 + schedule.OpenMin
		closeMinutes := schedule.CloseHour*60 + schedule.CloseMin + 24*60 // Add 24 hours

		// Adjust current time if after midnight
		if currentMinutes < closeMinutes {
			currentMinutes += 24 * 60
		}

		return currentMinutes >= openMinutes && currentMinutes < closeMinutes
	}

	openMinutes := schedule.OpenHour*60 + schedule.OpenMin
	closeMinutes := schedule.CloseHour*60 + schedule.CloseMin
	return currentMinutes >= openMinutes && currentMinutes < closeMinutes
}
