package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var planOrder = map[string]int{
	"free":  0,
	"pro":   1,
	"chain": 2,
}

// RequirePlan returns a middleware that enforces a minimum subscription plan.
func RequirePlan(minPlan string) gin.HandlerFunc {
	return func(c *gin.Context) {
		plan, exists := c.Get("plan")
		if !exists {
			plan = "free"
		}

		userPlanStr, _ := plan.(string)
		userLevel := planOrder[userPlanStr]
		minLevel := planOrder[minPlan]

		if userLevel < minLevel {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":        "plan upgrade required",
				"required_plan": minPlan,
				"current_plan": userPlanStr,
			})
			return
		}
		c.Next()
	}
}
