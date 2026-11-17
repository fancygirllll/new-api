package controller

import (
	"github.com/gin-gonic/gin"
)

// VideoGenerations
// @Summary Generate video
// @Description Generate a video via the video generation API
// @Description Supports multiple video generation services:
// @Description - Kling AI: https://app.klingai.com/cn/dev/document-api/apiReference/commonInfo
// @Description - Jimeng: https://www.volcengine.com/docs/85621/1538636
// @Tags Video
// @Accept json
// @Produce json
// @Param Authorization header string true "User authentication token (Access-Token: sk-xxxx)"
// @Param request body dto.VideoRequest true "Video generation request parameters"
// @Failure 400 {object} dto.OpenAIError "Invalid request parameters"
// @Failure 401 {object} dto.OpenAIError "Unauthorized"
// @Failure 403 {object} dto.OpenAIError "Forbidden"
// @Failure 500 {object} dto.OpenAIError "Internal server error"
// @Router /v1/video/generations [post]
func VideoGenerations(c *gin.Context) {
}

// VideoGenerationsTaskId
// @Summary Query video
// @Description Query the status and result of a video generation task by task ID
// @Tags Video
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param task_id path string true "Task ID"
// @Success 200 {object} dto.VideoTaskResponse "Task status and result"
// @Failure 400 {object} dto.OpenAIError "Invalid request parameters"
// @Failure 401 {object} dto.OpenAIError "Unauthorized"
// @Failure 403 {object} dto.OpenAIError "Forbidden"
// @Failure 500 {object} dto.OpenAIError "Internal server error"
// @Router /v1/video/generations/{task_id} [get]
func VideoGenerationsTaskId(c *gin.Context) {
}

// KlingText2VideoGenerations
// @Summary Kling text-to-video
// @Description Call the Kling AI text-to-video API to generate video content
// @Tags Video
// @Accept json
// @Produce json
// @Param Authorization header string true "User authentication token (Access-Token: sk-xxxx)"
// @Param request body KlingText2VideoRequest true "Video generation request parameters"
// @Success 200 {object} dto.VideoTaskResponse "Task status and result"
// @Failure 400 {object} dto.OpenAIError "Invalid request parameters"
// @Failure 401 {object} dto.OpenAIError "Unauthorized"
// @Failure 403 {object} dto.OpenAIError "Forbidden"
// @Failure 500 {object} dto.OpenAIError "Internal server error"
// @Router /kling/v1/videos/text2video [post]
func KlingText2VideoGenerations(c *gin.Context) {
}

type KlingText2VideoRequest struct {
	ModelName      string              `json:"model_name,omitempty" example:"kling-v1"`
	Prompt         string              `json:"prompt" binding:"required" example:"A cat playing piano in the garden"`
	NegativePrompt string              `json:"negative_prompt,omitempty" example:"blurry, low quality"`
	CfgScale       float64             `json:"cfg_scale,omitempty" example:"0.7"`
	Mode           string              `json:"mode,omitempty" example:"std"`
	CameraControl  *KlingCameraControl `json:"camera_control,omitempty"`
	AspectRatio    string              `json:"aspect_ratio,omitempty" example:"16:9"`
	Duration       string              `json:"duration,omitempty" example:"5"`
	CallbackURL    string              `json:"callback_url,omitempty" example:"https://your.domain/callback"`
	ExternalTaskId string              `json:"external_task_id,omitempty" example:"custom-task-001"`
}

type KlingCameraControl struct {
	Type   string             `json:"type,omitempty" example:"simple"`
	Config *KlingCameraConfig `json:"config,omitempty"`
}

type KlingCameraConfig struct {
	Horizontal float64 `json:"horizontal,omitempty" example:"2.5"`
	Vertical   float64 `json:"vertical,omitempty" example:"0"`
	Pan        float64 `json:"pan,omitempty" example:"0"`
	Tilt       float64 `json:"tilt,omitempty" example:"0"`
	Roll       float64 `json:"roll,omitempty" example:"0"`
	Zoom       float64 `json:"zoom,omitempty" example:"0"`
}

// KlingImage2VideoGenerations
// @Summary Kling official - image-to-video
// @Description Call the Kling AI image-to-video API to generate video content
// @Tags Video
// @Accept json
// @Produce json
// @Param Authorization header string true "User authentication token (Access-Token: sk-xxxx)"
// @Param request body KlingImage2VideoRequest true "Image-to-video request parameters"
// @Success 200 {object} dto.VideoTaskResponse "Task status and result"
// @Failure 400 {object} dto.OpenAIError "Invalid request parameters"
// @Failure 401 {object} dto.OpenAIError "Unauthorized"
// @Failure 403 {object} dto.OpenAIError "Forbidden"
// @Failure 500 {object} dto.OpenAIError "Internal server error"
// @Router /kling/v1/videos/image2video [post]
func KlingImage2VideoGenerations(c *gin.Context) {
}

type KlingImage2VideoRequest struct {
	ModelName      string              `json:"model_name,omitempty" example:"kling-v2-master"`
	Image          string              `json:"image" binding:"required" example:"https://h2.inkwai.com/bs2/upload-ylab-stunt/se/ai_portal_queue_mmu_image_upscale_aiweb/3214b798-e1b4-4b00-b7af-72b5b0417420_raw_image_0.jpg"`
	Prompt         string              `json:"prompt,omitempty" example:"A cat playing piano in the garden"`
	NegativePrompt string              `json:"negative_prompt,omitempty" example:"blurry, low quality"`
	CfgScale       float64             `json:"cfg_scale,omitempty" example:"0.7"`
	Mode           string              `json:"mode,omitempty" example:"std"`
	CameraControl  *KlingCameraControl `json:"camera_control,omitempty"`
	AspectRatio    string              `json:"aspect_ratio,omitempty" example:"16:9"`
	Duration       string              `json:"duration,omitempty" example:"5"`
	CallbackURL    string              `json:"callback_url,omitempty" example:"https://your.domain/callback"`
	ExternalTaskId string              `json:"external_task_id,omitempty" example:"custom-task-002"`
}

// KlingImage2videoTaskId godoc
// @Summary Kling task query - image-to-video
// @Description Query the status and result of a Kling video generation task by task ID
// @Tags Origin
// @Accept json
// @Produce json
// @Param task_id path string true "Task ID"
// @Router /kling/v1/videos/image2video/{task_id} [get]
func KlingImage2videoTaskId(c *gin.Context) {}

// KlingText2videoTaskId godoc
// @Summary Kling task query - text-to-video
// @Description Query the status and result of a Kling text-to-video generation task by task ID
// @Tags Origin
// @Accept json
// @Produce json
// @Param task_id path string true "Task ID"
// @Router /kling/v1/videos/text2video/{task_id} [get]
func KlingText2videoTaskId(c *gin.Context) {}
