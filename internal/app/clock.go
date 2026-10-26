package app

import (
	"fmt"
	"time"
)

// One authoritative server instant per turn. The browser supplies a timezone,
// never the date or clock, so relative dates cannot drift between research steps.
func (a *App) turnClock(s *Session) time.Time {
	if s.TurnTime.IsZero() {
		s.TurnTime = time.Now()
		if a.clock != nil {
			s.TurnTime = a.clock()
		}
	}
	t := s.TurnTime
	if loc, err := time.LoadLocation(s.SearchTimeZone); s.SearchTimeZone != "" && err == nil {
		return t.In(loc)
	}
	return t.In(time.Local)
}

func (a *App) clockInstructions(s *Session) string {
	t := a.turnClock(s)
	return fmt.Sprintf("Authoritative current date and time for this turn: %s; timezone %s; UTC %s. Today is %s; yesterday is %s; tomorrow is %s. Resolve relative dates in this timezone, including daylight-saving changes. A question without a date does not mean today: 'the score/result' usually asks for the latest completed event. Do not confuse publication dates with event dates or absence of evidence with evidence that no event occurred.", t.Format("Monday, 2 January 2006 15:04:05 -07:00"), t.Location(), t.UTC().Format(time.RFC3339), t.Format("2006-01-02"), t.AddDate(0, 0, -1).Format("2006-01-02"), t.AddDate(0, 0, 1).Format("2006-01-02"))
}
