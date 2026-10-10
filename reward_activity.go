package main

import "log"

// recordRewardActivity records a potential reward for later eligibility checks.
// It does not calculate money or trigger a payout.
func recordRewardActivity(userID int, postID string, activityType string, activityKey string) {
	_, err := db.Exec(`
		INSERT INTO reward_activity (
			user_id,
			post_id,
			activity_type,
			activity_key,
			quantity,
			reward_month,
			status
		)
		VALUES (
			$1,
			NULLIF($2, '')::integer,
			$3,
			$4 || ':' || TO_CHAR(CURRENT_TIMESTAMP, 'YYYY-MM'),
			1,
			DATE_TRUNC('month', CURRENT_TIMESTAMP)::date,
			'pending'
		)
		ON CONFLICT (user_id, activity_type, activity_key) DO NOTHING
	`, userID, postID, activityType, activityKey)

	if err != nil {
		log.Println("Unable to record pending reward activity:", err)
	}
}

// recordVideoWatchActivity records video watch time for later eligibility checks.
// It does not calculate money or trigger a payout.
func recordVideoWatchActivity(userID int, postID int, watchedSeconds int) {
	_, err := db.Exec(`
		INSERT INTO reward_activity (
			user_id,
			post_id,
			activity_type,
			activity_key,
			quantity,
			reward_month,
			status
		)
		VALUES (
			$1,
			$2,
			'video_watch',
			'post:' || $2::text || ':' || TO_CHAR(CURRENT_TIMESTAMP, 'YYYY-MM'),
			$3,
			DATE_TRUNC('month', CURRENT_TIMESTAMP)::date,
			'pending'
		)
		ON CONFLICT (user_id, activity_type, activity_key) DO UPDATE SET quantity = reward_activity.quantity + EXCLUDED.quantity WHERE reward_activity.status = 'pending'
	`, userID, postID, watchedSeconds)

	if err != nil {
		log.Println("Unable to record pending video-watch activity:", err)
	}
}
