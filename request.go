package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type ImgDataJSON struct {
	Created int `json:"created"`
	Data    []struct {
		Url string `json:"url"`
	} `json:"data"`
}

var tokenAPI string = "*****YOUR TOKEN*******"

func main() {
	date := time.Now().UTC().Format("2 January")
	clientList := openai.NewClient(tokenAPI)
	resp, err := clientList.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4Turbo1106,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Show the list of holydays in " + date + ". You can use information from past years.",
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}
	hdayList := resp.Choices[0].Message.Content
	fmt.Println(resp.Choices[0].Message.Content)
	clientPromt := openai.NewClient(tokenAPI)
	resp, err = clientPromt.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     openai.GPT4Turbo1106,
			MaxTokens: 200,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Show me only short promt with maximum length of 125 symbols to create image in realistic style with elements of these holydays:\n" + hdayList,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}
	hdayPrompt := resp.Choices[0].Message.Content
	fmt.Println(resp.Choices[0].Message.Content)

	url := "https://api.openai.com/v1/images/generations"
	hdayPrompt = strings.ReplaceAll(hdayPrompt, "\"", "")
	payload := []byte(`{
		"model": "dall-e-3",
		"prompt": "` + hdayPrompt + `",
		"n": 1,
		"size": "1024x1024"
	}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenAPI)

	client := &http.Client{}
	respImg, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer respImg.Body.Close()

	fmt.Println("Response Status Code:", respImg.Status)
	fmt.Println(respImg.Body)
	imgBytes, err := io.ReadAll(respImg.Body)
	if err != nil {
		fmt.Println("Error reading response respImg:", err)

	}
	var dalleJSON ImgDataJSON
	json.Unmarshal(imgBytes, &dalleJSON)
	fmt.Println(dalleJSON.Created)
	filename := date + ".png"
	fmt.Println(dalleJSON)
	loadedImage, err := loadImageFromURL(dalleJSON.Data[0].Url)
	if err != nil {
		fmt.Println("Error loading message:", err)

	}
	target := image.NewRGBA(image.Rect(0, 0, 1024, 1024))

	// Draw a white layer.
	draw.Draw(target, target.Bounds(), image.White, image.ZP, draw.Src)

	// Draw the child image.
	draw.Draw(target, loadedImage.Bounds(), loadedImage, image.Point{0, 0}, draw.Src)

	// Encode to JPEG.
	var imageBuf bytes.Buffer
	err = png.Encode(&imageBuf, target)
	if err != nil {
		log.Panic(err)
	}

	// Write to a file.
	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	fw := bufio.NewWriter(fo)
	fw.Write(imageBuf.Bytes())
	fmt.Println("The image was saved as " + filename)
}

func loadImageFromURL(URL string) (image.Image, error) {
	// Get the response bytes from the URL
	response, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, errors.New("received non-200 response code")
	}

	img, _, err := image.Decode(response.Body)
	if err != nil {
		return nil, err
	}

	return img, nil
}
