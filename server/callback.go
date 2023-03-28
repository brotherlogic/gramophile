package server




func(s *Server) (w http.ResponseWriter, r *http.Request) {
	d := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))
	
})