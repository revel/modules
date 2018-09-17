package model

import "time"

// The file information contains information about the file that will be rendered on the page
type FileInformation struct {
	Icon     string     // The icon
	Name     string     // The file name
	Relative string     // Relative path to current request
	Size     int64      // The size of the file
	NiceSize string     // The size of the file
	SizeType string     // The type of size
	Modified *time.Time // The last modified date
}
