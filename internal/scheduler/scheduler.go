package scheduler

import (
	"log"
	"time"

	"github.com/brkss/dextrace/internal/domain"
	"github.com/brkss/dextrace/internal/usecase"
)

type Scheduler struct {
	sibionicUseCase   *usecase.SibionicUseCase
	nightscoutUseCase *usecase.NightscoutUsecase
	userID            string
	user              domain.User
	stopChan          chan bool
}

func NewScheduler(sibionicUseCase *usecase.SibionicUseCase, nightscoutUseCase *usecase.NightscoutUsecase, userID string, user domain.User) *Scheduler {
	return &Scheduler{
		sibionicUseCase:   sibionicUseCase,
		nightscoutUseCase: nightscoutUseCase,
		userID:            userID,
		user:              user,
		stopChan:          make(chan bool),
	}
}

func (s *Scheduler) Start() {
	log.Println("Starting scheduler for push-to-nightscout (5 minutes after last Nightscout record)")

	// Run immediately on start
	s.pushToNightscout()

	// Schedule next push based on last record
	s.scheduleNextPush()
}

func (s *Scheduler) scheduleNextPush() {
	for {
		select {
		case <-s.stopChan:
			log.Println("Scheduler stopped")
			return
		default:
			// Get the last record from Nightscout
			lastRecord, err := s.nightscoutUseCase.GetLastRecord()
			if err != nil {
				log.Printf("Error getting last record from Nightscout: %v", err)
				// If we can't get the last record, wait 5 minutes and try again
				time.Sleep(5 * time.Minute)
				continue
			}

			var nextPushTime time.Time

			if lastRecord == nil {
				// If Nightscout reports no records, avoid an immediate second push.
				// Pushes already happen on startup and after each wait interval.
				log.Println("No records found in Nightscout, scheduling next push in 5 minutes")
				nextPushTime = time.Now().Add(5 * time.Minute)
			} else {
				// Calculate time based on last record
				lastRecordTime := time.Unix(lastRecord.Date/1000, 0) // Convert milliseconds to seconds
				nextPushTime = lastRecordTime.Add(5 * time.Minute)

				// If the calculated time is in the past, push immediately
				if nextPushTime.Before(time.Now()) {
					log.Printf("Last record is older than 5 minutes, pushing immediately")
					s.pushToNightscout()
					nextPushTime = time.Now().Add(5 * time.Minute)
				} else {
					log.Printf("Scheduling next push at %v (5 minutes after last record at %v)",
						nextPushTime.Format("2006-01-02 15:04:05"),
						lastRecordTime.Format("2006-01-02 15:04:05"))
				}
			}

			// Wait until next push time
			waitDuration := time.Until(nextPushTime)
			if waitDuration > 0 {
				log.Printf("Waiting %v until next push", waitDuration)
				select {
				case <-time.After(waitDuration):
					s.pushToNightscout()
				case <-s.stopChan:
					log.Println("Scheduler stopped")
					return
				}
			}
		}
	}
}

func (s *Scheduler) Stop() {
	s.stopChan <- true
}

func (s *Scheduler) pushToNightscout() {
	log.Println("Executing scheduled push-to-nightscout")

	data, err := s.sibionicUseCase.GetGlucoseData(s.user, s.userID)
	if err != nil {
		log.Printf("Error getting glucose data: %v", err)
		return
	}

	err = s.nightscoutUseCase.PushData(*data)
	if err != nil {
		log.Printf("Error pushing data to Nightscout: %v", err)
		return
	}

	log.Println("Successfully pushed data to Nightscout")
}
