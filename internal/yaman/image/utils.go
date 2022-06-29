package image

// isNameValid validates the name of an image.
func isNameValid(name string) bool {
	return imageNamePattern.MatchString(name)
}
