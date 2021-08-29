package cliwa

import (
	"github.com/jypelle/mifasol/restApiV1"
	"html"
	"net/url"
	"strconv"
	"syscall/js"
)

func (c *App) retrieveServerCredentials() {
	rawUrl := js.Global().Get("window").Get("location").Get("href").String()
	baseUrl, _ := url.Parse(rawUrl)

	c.config.ServerHostname = baseUrl.Hostname()
	c.config.ServerPort, _ = strconv.ParseInt(baseUrl.Port(), 10, 64)
	c.config.ServerSsl = baseUrl.Scheme == "https"
}

func (c *App) showStartComponent() {
	c.restClient = nil
	c.localDb = nil

	body := c.doc.Get("body")
	body.Set("innerHTML", c.RenderTemplate(nil, "start.html"))

	// Set focus
	c.doc.Call("getElementById", "mifasolUsername").Call("focus")

	// Set button
	js.Global().Set("logInAction", js.FuncOf(c.logInAction))
}

func (c *App) showHomeComponent() {
	body := c.doc.Get("body")
	body.Set("innerHTML", c.RenderTemplate(nil, "home.html"))

	// Set callback
	js.Global().Set("logOutAction", js.FuncOf(c.logOutAction))
	js.Global().Set("refreshAction", js.FuncOf(c.refreshAction))
	js.Global().Set("showLibraryArtistsAction", js.FuncOf(c.showLibraryArtistsAction))
	js.Global().Set("showLibraryAlbumsAction", js.FuncOf(c.showLibraryAlbumsAction))
	js.Global().Set("showLibrarySongsAction", js.FuncOf(c.showLibrarySongsAction))
	js.Global().Set("showLibraryPlaylistsAction", js.FuncOf(c.showLibraryPlaylistsAction))
	js.Global().Set("playSongAction", js.FuncOf(c.playSongAction))
	js.Global().Set("openArtistAction", js.FuncOf(c.openArtistAction))
	js.Global().Set("openAlbumAction", js.FuncOf(c.openAlbumAction))
	js.Global().Set("openPlaylistAction", js.FuncOf(c.openPlaylistAction))

	go func() {
		c.Refresh()
		c.showLibraryArtistsComponent()
	}()

}

func (c *App) showLibraryArtistsComponent() {
	listDiv := c.doc.Call("getElementById", "libraryList")

	var divContent string
	for _, artist := range c.localDb.OrderedArtists {
		if artist == nil {
			divContent += `<div class="artistItem"><a class="artistLink" href="#" onclick="openArtistAction(this.getAttribute('data-artistId'));return false;" data-artistId="` + string(restApiV1.UnknownArtistId) + `">(Unknown artist)</a></div>`
		} else {
			divContent += `<div class="artistItem"><a class="artistLink" href="#" onclick="openArtistAction(this.getAttribute('data-artistId'));return false;" data-artistId="` + string(artist.Id) + `">` + html.EscapeString(artist.Name) + `</a></div>`
		}
	}
	listDiv.Set("innerHTML", divContent)
}

func (c *App) showLibraryAlbumsComponent() {
	listDiv := c.doc.Call("getElementById", "libraryList")

	var divContent string

	for _, album := range c.localDb.OrderedAlbums {
		if album == nil {
			divContent += `<div class="albumItem"><a class="albumLink" href="#" onclick="openAlbumAction(this.getAttribute('data-albumId'));return false;" data-albumId="` + string(restApiV1.UnknownAlbumId) + `">(Unknown album)</a>`
		} else {
			divContent += `<div class="albumItem"><a class="albumLink" href="#" onclick="openAlbumAction(this.getAttribute('data-albumId'));return false;" data-albumId="` + string(album.Id) + `">` + html.EscapeString(album.Name) + `</a>`

			if len(album.ArtistIds) > 0 {
				for _, artistId := range album.ArtistIds {
					divContent += ` / <a class="artistLink" href="#" onclick="openArtistAction(this.getAttribute('data-artistId'));return false;" data-artistId="` + string(artistId) + `">` + html.EscapeString(c.localDb.Artists[artistId].Name) + `</a>`
				}
			}
		}
		divContent += `</div>`
	}
	listDiv.Set("innerHTML", divContent)
}

func (c *App) showLibrarySongsComponent(artistId *restApiV1.ArtistId, albumId *restApiV1.AlbumId, playlistId *restApiV1.PlaylistId) {
	listDiv := c.doc.Call("getElementById", "libraryList")

	var divContent string
	var songList []*restApiV1.Song

	if playlistId == nil {
		if artistId != nil {
			songList = c.localDb.ArtistOrderedSongs[*artistId]
		} else if albumId != nil {
			songList = c.localDb.AlbumOrderedSongs[*albumId]
		} else {
			songList = c.localDb.OrderedSongs
		}

		for _, song := range songList {
			divContent += c.showSongItem(song)
		}
	} else {
		for _, songId := range c.localDb.Playlists[*playlistId].SongIds {
			divContent += c.showSongItem(c.localDb.Songs[songId])
		}
	}

	listDiv.Set("innerHTML", divContent)
}

func (c *App) showSongItem(song *restApiV1.Song) string {
	var divContent = `<div class="songItem">`
	divContent += `<a class="songLink" href="#" onclick="playSongAction(this.getAttribute('data-songId'));return false;" data-songId="` + string(song.Id) + `">` + html.EscapeString(song.Name) + `</a>`

	if song.AlbumId != restApiV1.UnknownAlbumId {
		divContent += ` / <a class="albumLink" href="#" onclick="openAlbumAction(this.getAttribute('data-albumId'));return false;" data-albumId="` + string(song.AlbumId) + `">` + html.EscapeString(c.localDb.Albums[song.AlbumId].Name) + `</a>`
	}

	if len(song.ArtistIds) > 0 {
		for _, artistId := range song.ArtistIds {
			divContent += ` / <a class="artistLink" href="#" onclick="openArtistAction(this.getAttribute('data-artistId'));return false;" data-artistId="` + string(artistId) + `">` + html.EscapeString(c.localDb.Artists[artistId].Name) + `</a>`
		}
	}

	divContent += `</div>`

	return divContent
}

func (c *App) showLibraryPlaylistsComponent() {
	listDiv := c.doc.Call("getElementById", "libraryList")

	var divContent string
	for _, playlist := range c.localDb.OrderedPlaylists {
		if playlist != nil {
			divContent += `<div class="playlistItem"><a class="playlistLink" href="#" onclick="openPlaylistAction(this.getAttribute('data-playlistId'));return false;" data-playlistId="` + string(playlist.Id) + `">` + html.EscapeString(playlist.Name) + `</a></div>`
		}
	}
	listDiv.Set("innerHTML", divContent)
}
