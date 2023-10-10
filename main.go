package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {

	engine := html.New("./templates", ".tpl")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", nil)
	})

	// Define a route for streaming video
	app.Get("/stream", streamVideo)
	app.Listen(":3000")

}
func streamVideo(c *fiber.Ctx) error {

	filePath := "video.mp4"

	// Open the video file
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening video file:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("Error getting file information:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	// get the file mime informations
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))

	// get file size
	fileSize := fileInfo.Size()

	// Get the range header from the request
	rangeHeader := c.GetReqHeaders()["range"]
	if rangeHeader != "" {
		var start, end int64

		ranges := strings.Split(rangeHeader, "=")
		if len(ranges) != 2 {
			log.Println("Invalid Range Header:", err)
			return c.Status(http.StatusInternalServerError).SendString("Internal Server Error")
		}

		byteRange := ranges[1]
		byteRanges := strings.Split(byteRange, "-")

		// get the start range
		start, err := strconv.ParseInt(byteRanges[0], 10, 64)
		if err != nil {
			log.Println("Error parsing start byte position:", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		// Calculate the end range
		if len(byteRanges) > 1 && byteRanges[1] != "" {
			end, err = strconv.ParseInt(byteRanges[1], 10, 64)
			if err != nil {
				log.Println("Error parsing end byte position:", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
			}
		} else {
			end = fileSize - 1
		}

		// Setting required response headers
		c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size())) // Set the Content-Range header
		c.Set(fiber.HeaderContentLength, strconv.FormatInt(end-start+1, 10))                        // Set the Content-Length header for the range being served
		c.Set(fiber.HeaderContentType, mimeType)                                                    // Set the Content-Type
		c.Set(fiber.HeaderAcceptRanges, "bytes")                                                    // Set Accept-Ranges
		c.Status(fiber.StatusPartialContent)                                                        // Set the status code to 206 (Partial Content)

		// Seek to the start position
		_, seekErr := file.Seek(start, io.SeekStart)
		if seekErr != nil {
			log.Println("Error seeking to start position:", seekErr)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		// Copy the specified range of bytes to the response
		_, copyErr := io.CopyN(c.Response().BodyWriter(), file, end-start+1)
		if copyErr != nil {
			log.Println("Error copying bytes to response:", copyErr)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

	} else {
		// If no Range header is present, serve the entire video
		c.Set("Content-Length", strconv.FormatInt(fileSize, 10))
		_, copyErr := io.Copy(c.Response().BodyWriter(), file)
		if copyErr != nil {
			log.Println("Error copying entire file to response:", copyErr)
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}
	}

	return nil
}
