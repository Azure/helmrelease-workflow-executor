package keptn

func resourceNameToURI(fname string) string {
	switch fname {
	case sliFilename:
		return sliURI
	case sloFilename:
		return sloURI
	case jobExecutorFilename:
		return jobExecutorURI
	default:
		return fname
	}
}
