package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Asset represents an asset in a GitHub release
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func main() {
	// Specify the owner and repository name
	owner := "GloriousEggroll"
	repo := "proton-ge-custom"

	// Specify the compatibility tools folder path
	steamCompatabilityFolderPath := "~/.steam/steam/compatibilitytools.d/"

	// Resolve the home directory and compatibility folder path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	steamCompatabilityFolderPath = filepath.Join(homeDir, steamCompatabilityFolderPath[2:])

	// Check if compatibility tools folder exists, if not create it.

	if _, err := os.Stat(steamCompatabilityFolderPath); os.IsNotExist(err) {
		if err != nil {
			err := os.Mkdir(steamCompatabilityFolderPath, 0755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not find or create: %v\n", steamCompatabilityFolderPath)
				os.Exit(1)
			}
		}
	}

	// Get the latest release tag and download URL of the .tar.gz file
	tagName, tarGzDownloadURL, err := getLatestReleaseURL(owner, repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching latest release information: %v\n", err)
		os.Exit(1)
	}

	GEFilePathWithLatestTagName := filepath.Join(steamCompatabilityFolderPath, tagName)

	if _, err := os.Stat(GEFilePathWithLatestTagName); err == nil {
		fmt.Printf("The latest version of GE-Proton, %s, is already installed. Exiting...\n", tagName)
		os.Exit(0)
	}

	// Determine the output path for the .tar.gz file in the compatibility folder
	newGEFileNameTarGz := fmt.Sprintf("GE-Proton-%s.tar.gz", tagName)
	newGEFTarGzFolderPath := filepath.Join(steamCompatabilityFolderPath, newGEFileNameTarGz)
	// Download the .tar.gz file
	fmt.Printf("Downloading .tar.gz file from %s to %s...\n", tarGzDownloadURL, newGEFTarGzFolderPath)
	err = downloadLatestRelease(tarGzDownloadURL, newGEFTarGzFolderPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading .tar.gz file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(".tar.gz file downloaded successfully to %s\n", newGEFTarGzFolderPath)

	// Extract the .tar.gz file to the specified folder
	fmt.Printf("Extracting %s to %s...\n", newGEFTarGzFolderPath, steamCompatabilityFolderPath)
	err = extractTarGzFile(newGEFTarGzFolderPath, steamCompatabilityFolderPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting .tar.gz file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Extraction successful!")

	// Remove the .tar.gz file after extraction is complete
	fmt.Printf("Removing the .tar.gz file %s...\n", newGEFTarGzFolderPath)
	err = os.Remove(newGEFTarGzFolderPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing .tar.gz file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(".tar.gz file removed successfully.")
}

// Function to get the latest release URL and download URL of the tar.gz file from a GitHub repository
func getLatestReleaseURL(owner, repo string) (string, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	// Make an HTTP GET request to the API URL
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var release GitHubRelease
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		return "", "", err
	}

	// Find the tar.gz file in the assets
	var tarGzDownloadURL string
	for _, asset := range release.Assets {
		if filepath.Ext(asset.Name) == ".gz" {
			tarGzDownloadURL = asset.DownloadURL
			break
		}
	}

	if tarGzDownloadURL == "" {
		return "", "", fmt.Errorf("no .tar.gz file found in the latest release")
	}

	// Return the latest release URL and the download URL for the tar.gz file
	return release.TagName, tarGzDownloadURL, nil
}

// Function to download the latest release file
func downloadLatestRelease(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the output file
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy data from the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Function to extract a .tar.gz file to a specified folder path
func extractTarGzFile(tarGzPath, outputFolderPath string) error {
	// Open the tar.gz file
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract files from the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// Reached the end of the archive
			break
		}
		if err != nil {
			return err
		}

		// Determine the output path for the extracted file
		outputPath := filepath.Join(outputFolderPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create a directory
			err = os.MkdirAll(outputPath, os.ModePerm)
			if err != nil {
				return err
			}

		case tar.TypeReg:
			// Create a file and write data
			outFile, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
