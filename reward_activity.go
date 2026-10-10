package main

import "log"

// recordRewardActivity records potential activity for later review.
// It does not calculate money or trigger a payout.
func recordRewardActivity(userID int, postID string, activityType string, activityKey string) {
	if activityType != "like" && activityType != "comment" {
		return
	}

	_, err := db.Exec(`
		INSERT INTO reward_activity (
			user_id, creator_id, post_id, activity_type, activity_key,
			quantity, reward_month, status
		)
		SELECT
			$1,
			p.user_id,
			p.id,
			$3,
			$4 || ':' || TO_CHAR(CURRENT_TIMESTAMP, 'YYYY-MM'),
			1,
			DATE_TRUNC('month', CURRENT_TIMESTAMP)::date,
			CASE
				WHEN EXISTS (
					SELECT 1
					FROM creator_monetization_applications ca
					WHERE ca.user_id = p.user_id
					  AND ca.status = 'approved'
				) THEN 'pending'
				ELSE 'rejected'
			END
		FROM posts p
		WHERE p.id = NULLIF($2, '')::integer
		  AND p.user_id <> $1
		ON CONFLICT (user_id, activity_type, activity_key) DO NOTHING
	`, userID, postID, activityType, activityKey)

	if err != nil {
		log.Println("Unable to record pending reward activity:", err)
	}
}
