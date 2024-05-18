package routes

import (
	"math/rand"
	"os"
	"strconv"
	"time"
	"url-shortener/database"
	"url-shortener/helpers"

	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

type request struct {
	URL         string
	CustomShort string
}

type response struct {
	URL             string
	CustomShort     string
	XRateRemaining  int
	XRateLimitReset time.Duration
}

func Shorten(ctx *fiber.Ctx) error {
	body := &request{}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	r2 := database.CreateClient(1)
	defer r2.Close()

	val, err := r2.Get(database.Ctx, ctx.IP()).Result()
	limit, _ := r2.TTL(database.Ctx, ctx.IP()).Result()

	if err == redis.Nil {
		// set quota for the current IP Address
		_ = r2.Set(database.Ctx, ctx.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else if err == nil {
		valInt, _ := strconv.Atoi(val)
		// If Quota exceeded
		if valInt <= 0 {
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":            "Rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}

	// check if the input is an actual url

	if !govalidator.IsURL(body.URL) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid URL"})
	}

	if !helpers.RemoveDomainError(body.URL) {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Can't do that"})
	}

	// enforce HTTPS, SSL
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string
	if body.CustomShort == "" {
		id = helpers.Base62Encode(rand.Uint64())
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()

	val, _ = r.Get(database.Ctx, id).Result()

	if val != "" {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "URL Custom short is already in use",
		})
	}

	err = r.Set(database.Ctx, id, body.URL, 0).Err()

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect to server",
		})
	}

	defaultAPIQuotaStr := os.Getenv("API_QUOTA")
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to connect to server",
		})
	}
	defaultApiQuota, _ := strconv.Atoi(defaultAPIQuotaStr)
	resp := response{
		URL:             body.URL,
		CustomShort:     "",
		XRateRemaining:  defaultApiQuota,
		XRateLimitReset: 30,
	}

	remainingQuota, err := r2.Decr(database.Ctx, ctx.IP()).Result()

	resp.XRateRemaining = int(remainingQuota)
	resp.XRateRemaining = int(limit / time.Nanosecond / time.Minute)

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return ctx.Status(fiber.StatusOK).JSON(resp)
}
