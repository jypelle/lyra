package ui

import (
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/jypelle/mifasol/primitive"
	"github.com/jypelle/mifasol/restApiV1"
	"github.com/rivo/tview"
	"strconv"
)

type LibraryComponent struct {
	*tview.Flex
	title         *tview.TextView
	list          *primitive.RichList
	uiApp         *UIApp
	historyFilter []*libraryFilter
	songs         []*restApiV1.Song
	albums        []*restApiV1.Album
	artists       []*restApiV1.Artist
	playlists     []*restApiV1.Playlist
}

type libraryFilter struct {
	libraryType libraryType
	artistId    *restApiV1.ArtistId
	albumId     *restApiV1.AlbumId
	playlistId  *restApiV1.PlaylistId
	userId      *restApiV1.UserId
	index       int
	position    int
}

type libraryType int64

const (
	libraryTypeMenu libraryType = iota
	libraryTypeAllArtists
	libraryTypeAllAlbums
	libraryTypeAllPlaylists
	libraryTypeAllSongs
	libraryTypeAllUsers
	libraryTypeSongsFromArtist
	libraryTypeSongsFromUnknownArtist
	libraryTypeSongsFromAlbum
	libraryTypeSongsFromUnknownAlbum
	libraryTypeSongsFromPlaylist
	libraryTypeFavoritePlaylistsFromUser
)

func (c libraryType) label() string {
	switch c {
	case libraryTypeMenu:
		return "Menu"
	case libraryTypeAllArtists:
		return "All artists"
	case libraryTypeAllAlbums:
		return "All albums"
	case libraryTypeAllPlaylists:
		return "All playlists"
	case libraryTypeAllSongs:
		return "All songs"
	case libraryTypeAllUsers:
		return "All users"
	case libraryTypeSongsFromArtist:
		return "Songs from %s artist"
	case libraryTypeSongsFromUnknownArtist:
		return "Songs from unknown artists"
	case libraryTypeSongsFromAlbum:
		return "Songs from %s album"
	case libraryTypeSongsFromUnknownAlbum:
		return "Songs from unknown albums"
	case libraryTypeSongsFromPlaylist:
		return "Songs from %s playlist"
	case libraryTypeFavoritePlaylistsFromUser:
		return "Favorite playlist from %s"
	}
	return ""
}

type libraryMenu int64

const (
	libraryMenuMyFavoritePlaylists libraryMenu = iota
	libraryMenuAllArtists
	libraryMenuAllAlbums
	libraryMenuAllPlaylists
	libraryMenuAllSongs
	libraryMenuAllUsers
)

func (c libraryMenu) label() string {
	switch c {
	case libraryMenuMyFavoritePlaylists:
		return "Favorite playlists"
	case libraryMenuAllArtists:
		return "All artists"
	case libraryMenuAllAlbums:
		return "All albums"
	case libraryMenuAllPlaylists:
		return "All playlists"
	case libraryMenuAllSongs:
		return "All songs"
	case libraryMenuAllUsers:
		return "All users"
	}
	return ""
}

var libraryMenus = []libraryMenu{
	libraryMenuMyFavoritePlaylists,
	libraryMenuAllArtists,
	libraryMenuAllAlbums,
	libraryMenuAllPlaylists,
	libraryMenuAllSongs,
	libraryMenuAllUsers,
}

func NewLibraryComponent(uiApp *UIApp) *LibraryComponent {

	c := &LibraryComponent{
		uiApp: uiApp,
	}

	c.title = tview.NewTextView()
	c.title.SetDynamicColors(true)
	c.title.SetText("[" + ColorTitleStr + "]💿 Library")

	c.list = primitive.NewRichList()
	c.list.SetInfiniteScroll(false)
	c.list.SetHighlightFullLine(true)
	c.list.SetPrefixWithLineNumber(true)
	c.list.SetSelectedBackgroundColor(ColorSelected)
	c.list.SetUnfocusedSelectedBackgroundColor(ColorUnfocusedSelected)
	c.list.SetBorder(false)

	c.list.SetHighlightedMainTextFunc(func(index int, mainText string) string {
		switch c.currentFilter().libraryType {
		case libraryTypeMenu:
			return libraryMenus[c.list.GetCurrentItem()].label()
		case libraryTypeAllArtists:
			return c.getMainTextArtist(c.uiApp.LocalDb().OrderedArtists[c.list.GetCurrentItem()], c.currentFilter().position)
		case libraryTypeAllAlbums:
			return c.getMainTextAlbum(c.uiApp.LocalDb().OrderedAlbums[c.list.GetCurrentItem()], c.currentFilter().position)
		case libraryTypeAllPlaylists:
			return c.getMainTextPlaylist(c.playlists[c.list.GetCurrentItem()], nil, c.currentFilter().position)
		case libraryTypeAllSongs:
			return c.getMainTextSong(c.songs[c.list.GetCurrentItem()], nil, nil, c.currentFilter().position)
		case libraryTypeAllUsers:
			return c.getMainTextUser(c.uiApp.LocalDb().OrderedUsers[c.list.GetCurrentItem()])
		case libraryTypeSongsFromAlbum:
			return c.getMainTextSong(c.songs[c.list.GetCurrentItem()], c.currentFilter().albumId, nil, c.currentFilter().position)
		case libraryTypeSongsFromUnknownAlbum:
			return c.getMainTextSong(c.songs[c.list.GetCurrentItem()], nil, nil, c.currentFilter().position)
		case libraryTypeSongsFromArtist:
			return c.getMainTextSong(c.songs[c.list.GetCurrentItem()], nil, c.currentFilter().artistId, c.currentFilter().position)
		case libraryTypeSongsFromUnknownArtist:
			return c.getMainTextSong(c.songs[c.list.GetCurrentItem()], nil, nil, c.currentFilter().position)
		case libraryTypeSongsFromPlaylist:
			return c.getMainTextSong(c.songs[c.list.GetCurrentItem()], nil, nil, c.currentFilter().position)
		case libraryTypeFavoritePlaylistsFromUser:
			return c.getMainTextPlaylist(c.uiApp.LocalDb().UserOrderedFavoritePlaylists[*c.currentFilter().userId][c.list.GetCurrentItem()], nil, c.currentFilter().position)
		}
		return ""
	})
	c.list.SetChangedFunc(func(index int, mainText string) {
		c.currentFilter().index = index
		c.currentFilter().position = 0
	})

	c.Flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(c.title, 1, 0, false).
		AddItem(c.list, 0, 1, false)

	c.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		currentFilter := c.currentFilter()

		switch {

		case event.Key() == tcell.KeyRune:
			switch event.Rune() {
			case 'a':
				if c.list.GetItemCount() > 0 && currentFilter.libraryType != libraryTypeMenu && currentFilter.libraryType != libraryTypeAllUsers {
					switch currentFilter.libraryType {
					case libraryTypeAllArtists:
						artist := c.uiApp.LocalDb().OrderedArtists[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().AddSongsFromArtist(artist)
					case libraryTypeAllAlbums:
						album := c.uiApp.LocalDb().OrderedAlbums[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().AddSongsFromAlbum(album)
					case libraryTypeAllPlaylists,
						libraryTypeFavoritePlaylistsFromUser:
						playlist := c.playlists[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().AddSongsFromPlaylist(playlist)
					case libraryTypeAllSongs,
						libraryTypeSongsFromAlbum,
						libraryTypeSongsFromUnknownAlbum,
						libraryTypeSongsFromArtist,
						libraryTypeSongsFromUnknownArtist,
						libraryTypeSongsFromPlaylist:
						song := c.songs[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().AddSong(song.Id)
					}
					c.list.SetCurrentItem(c.list.GetCurrentItem() + 1)
				}
				return nil

			case 'l':
				if c.list.GetItemCount() > 0 && currentFilter.libraryType != libraryTypeMenu && currentFilter.libraryType != libraryTypeAllUsers {
					switch currentFilter.libraryType {
					case libraryTypeAllArtists:
						artist := c.uiApp.LocalDb().OrderedArtists[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().LoadSongsFromArtist(artist)
					case libraryTypeAllAlbums:
						album := c.uiApp.LocalDb().OrderedAlbums[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().LoadSongsFromAlbum(album)
					case libraryTypeAllPlaylists,
						libraryTypeFavoritePlaylistsFromUser:
						playlist := c.playlists[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().LoadSongsFromPlaylist(playlist)
					case libraryTypeAllSongs,
						libraryTypeSongsFromArtist,
						libraryTypeSongsFromUnknownArtist,
						libraryTypeSongsFromAlbum,
						libraryTypeSongsFromUnknownAlbum,
						libraryTypeSongsFromPlaylist:
						song := c.songs[c.list.GetCurrentItem()]
						c.uiApp.CurrentComponent().LoadSong(song.Id)
					}
				}
				return nil

			case 'c':
				if currentFilter.libraryType != libraryTypeMenu {
					switch currentFilter.libraryType {
					case libraryTypeAllArtists:
						OpenArtistCreateComponent(c.uiApp, c)
					case libraryTypeAllAlbums:
						OpenAlbumCreateComponent(c.uiApp, c)
					case libraryTypeAllUsers:
						OpenUserCreateComponent(c.uiApp, c)
					}
				}
				return nil

			case 'd':
				if c.list.GetItemCount() > 0 && currentFilter.libraryType != libraryTypeMenu {
					switch currentFilter.libraryType {
					case libraryTypeAllArtists:
						artist := c.uiApp.LocalDb().OrderedArtists[c.list.GetCurrentItem()]
						if artist != nil {
							c.uiApp.ConfirmArtistDelete(artist)
						}
					case libraryTypeAllAlbums:
						album := c.uiApp.LocalDb().OrderedAlbums[c.list.GetCurrentItem()]
						if album != nil {
							c.uiApp.ConfirmAlbumDelete(album)
						}
					case libraryTypeAllPlaylists,
						libraryTypeFavoritePlaylistsFromUser:
						playlist := c.playlists[c.list.GetCurrentItem()]
						if playlist != nil {
							c.uiApp.ConfirmPlaylistDelete(playlist)
						}
					case libraryTypeAllUsers:
						user := c.uiApp.LocalDb().OrderedUsers[c.list.GetCurrentItem()]
						if user != nil {
							c.uiApp.ConfirmUserDelete(user)
						}
					case libraryTypeAllSongs,
						libraryTypeSongsFromAlbum,
						libraryTypeSongsFromUnknownAlbum,
						libraryTypeSongsFromArtist,
						libraryTypeSongsFromUnknownArtist,
						libraryTypeSongsFromPlaylist:
						song := c.songs[c.list.GetCurrentItem()]
						if song != nil {
							c.uiApp.ConfirmSongDelete(song)
						}
					}
				}
				return nil

			case 'e':
				if c.list.GetItemCount() > 0 && currentFilter.libraryType != libraryTypeMenu {
					switch currentFilter.libraryType {
					case libraryTypeAllArtists:
						artist := c.uiApp.LocalDb().OrderedArtists[c.list.GetCurrentItem()]
						if artist != nil {
							OpenArtistEditComponent(c.uiApp, artist.Id, &artist.ArtistMeta, c)
						}
					case libraryTypeAllAlbums:
						album := c.uiApp.LocalDb().OrderedAlbums[c.list.GetCurrentItem()]
						if album != nil {
							OpenAlbumEditComponent(c.uiApp, album.Id, &album.AlbumMeta, c)
						}
					case libraryTypeAllPlaylists,
						libraryTypeFavoritePlaylistsFromUser:
						playlist := c.playlists[c.list.GetCurrentItem()]
						if playlist != nil {
							OpenPlaylistEditComponent(c.uiApp, playlist, c)
						}
					case libraryTypeAllUsers:
						user := c.uiApp.LocalDb().OrderedUsers[c.list.GetCurrentItem()]
						if user != nil {
							OpenUserEditComponent(c.uiApp, user.Id, &user.UserMeta, c)
						}
					case libraryTypeAllSongs,
						libraryTypeSongsFromAlbum,
						libraryTypeSongsFromUnknownAlbum,
						libraryTypeSongsFromArtist,
						libraryTypeSongsFromUnknownArtist,
						libraryTypeSongsFromPlaylist:
						song := c.songs[c.list.GetCurrentItem()]
						if song != nil {
							OpenSongEditComponent(c.uiApp, song, c)
						}
					}
				}
				return nil

			case 'f':
				if c.list.GetItemCount() > 0 && currentFilter.libraryType != libraryTypeMenu {
					switch currentFilter.libraryType {
					case libraryTypeAllPlaylists,
						libraryTypeFavoritePlaylistsFromUser:
						playlist := c.playlists[c.list.GetCurrentItem()]
						if playlist != nil {
							myFavoritePlaylists := c.uiApp.LocalDb().UserFavoritePlaylists[c.uiApp.ConnectedUserId()]
							favoritePlaylistId := restApiV1.FavoritePlaylistId{
								UserId:     c.uiApp.ConnectedUserId(),
								PlaylistId: playlist.Id,
							}
							if _, ok := myFavoritePlaylists[playlist.Id]; ok {
								_, cliErr := c.uiApp.restClient.DeleteFavoritePlaylist(favoritePlaylistId)
								if cliErr != nil {
									c.uiApp.ClientErrorMessage("Unable to add playlist to favorites", cliErr)
								}
								c.uiApp.Reload()
							} else {
								_, cliErr := c.uiApp.restClient.CreateFavoritePlaylist(&restApiV1.FavoritePlaylistMeta{Id: favoritePlaylistId})
								if cliErr != nil {
									c.uiApp.ClientErrorMessage("Unable to remove playlist from favorites", cliErr)
								}
								c.uiApp.Reload()
							}
							if !(currentFilter.libraryType == libraryTypeFavoritePlaylistsFromUser && *currentFilter.userId == c.uiApp.ConnectedUserId()) {
								c.list.SetCurrentItem(c.list.GetCurrentItem() + 1)
							}
						}
					}
				}
				return nil

			}
		case event.Key() == tcell.KeyDEL, event.Key() == tcell.KeyBackspace:
			c.GoToPreviousFilter()
			return nil
		case event.Key() == tcell.KeyLeft:
			if currentFilter.position > 0 {
				currentFilter.position--
			}
		case event.Key() == tcell.KeyRight:
			if currentFilter.position < 30 {
				currentFilter.position++
			}
		case event.Key() == tcell.KeyEnter:
			if c.list.GetItemCount() > 0 {
				switch currentFilter.libraryType {
				case libraryTypeMenu:
					libraryMenu := libraryMenus[c.list.GetCurrentItem()]
					switch libraryMenu {
					case libraryMenuMyFavoritePlaylists:
						c.GoToFavoritePlaylistsFromUserFilter(c.uiApp.ConnectedUserId())
					case libraryMenuAllArtists:
						c.GoToAllArtistsFilter()
					case libraryMenuAllAlbums:
						c.GoToAllAlbumsFilter()
					case libraryMenuAllPlaylists:
						c.GoToAllPlaylistsFilter()
					case libraryMenuAllSongs:
						c.GoToAllSongsFilter()
					case libraryMenuAllUsers:
						c.GoToAllUsersFilter()
					}
				case libraryTypeAllArtists:
					artist := c.uiApp.LocalDb().OrderedArtists[c.list.GetCurrentItem()]
					if artist == nil {
						c.GoToSongsFromUnknownArtistFilter()
					} else {
						songId, artistId, albumId := c.getPositionnedIdArtist(c.uiApp.LocalDb().OrderedArtists[c.list.GetCurrentItem()], c.currentFilter().position)
						c.open(songId, artistId, albumId)
					}
				case libraryTypeAllAlbums:
					album := c.uiApp.LocalDb().OrderedAlbums[c.list.GetCurrentItem()]
					if album == nil {
						c.GoToSongsFromUnknownAlbumFilter()
					} else {
						songId, artistId, albumId := c.getPositionnedIdAlbum(c.uiApp.LocalDb().OrderedAlbums[c.list.GetCurrentItem()], c.currentFilter().position)
						c.open(songId, artistId, albumId)
					}
				case libraryTypeAllPlaylists,
					libraryTypeFavoritePlaylistsFromUser:
					playlist := c.playlists[c.list.GetCurrentItem()]
					c.GoToSongsFromPlaylistFilter(playlist.Id)
				case libraryTypeAllSongs,
					libraryTypeSongsFromAlbum,
					libraryTypeSongsFromUnknownAlbum,
					libraryTypeSongsFromArtist,
					libraryTypeSongsFromUnknownArtist,
					libraryTypeSongsFromPlaylist:

					songId, artistId, albumId := c.getPositionnedIdSong(c.songs[c.list.GetCurrentItem()], c.currentFilter().albumId, c.currentFilter().artistId, c.currentFilter().position)
					c.open(songId, artistId, albumId)
				case libraryTypeAllUsers:
				}
			}
			return nil

		}

		return event
	})

	// Start with menu
	c.ResetToMenuFilter()

	return c
}

func (c *LibraryComponent) Enable() {
	c.title.SetBackgroundColor(tcell.NewHexColor(0xe0e0e0))
	c.list.SetBackgroundColor(ColorEnabled)
}

func (c *LibraryComponent) Disable() {
	c.title.SetBackgroundColor(tcell.NewHexColor(0xc0c0c0))
	c.list.SetBackgroundColor(ColorDisabled)
}

func (c *LibraryComponent) getTitlePrefix() string {
	return "[" + ColorTitleStr + "]💿 Library"
}

func (c *LibraryComponent) currentFilter() *libraryFilter {
	return c.historyFilter[len(c.historyFilter)-1]
}

func (c *LibraryComponent) historizeLibraryFilter(newLibraryFilter *libraryFilter) {
	if len(c.historyFilter) > 0 {
		c.currentFilter().index = c.list.GetCurrentItem()
	}
	c.historyFilter = append(c.historyFilter, newLibraryFilter)
	c.RefreshView()
}

func (c *LibraryComponent) GoToPreviousFilter() {
	if len(c.historyFilter) > 1 {
		c.historyFilter = c.historyFilter[:len(c.historyFilter)-1]
	}
	c.RefreshView()
}

func (c *LibraryComponent) ResetToMenuFilter() {
	c.historyFilter = nil
	c.GoToMenuFilter()
}

func (c *LibraryComponent) GoToMenuFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeMenu})
}

func (c *LibraryComponent) GoToAllAlbumsFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeAllAlbums})
}

func (c *LibraryComponent) GoToAllArtistsFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeAllArtists})
}

func (c *LibraryComponent) GoToAllPlaylistsFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeAllPlaylists})
}

func (c *LibraryComponent) GoToAllSongsFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeAllSongs})
}

func (c *LibraryComponent) GoToAllUsersFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeAllUsers})
}

func (c *LibraryComponent) GoToSongsFromAlbumFilter(albumId restApiV1.AlbumId) {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeSongsFromAlbum, albumId: &albumId})
}

func (c *LibraryComponent) GoToSongsFromUnknownAlbumFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeSongsFromUnknownAlbum})
}

func (c *LibraryComponent) GoToSongsFromArtistFilter(artistId restApiV1.ArtistId) {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeSongsFromArtist, artistId: &artistId})
}

func (c *LibraryComponent) GoToSongsFromUnknownArtistFilter() {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeSongsFromUnknownArtist})
}

func (c *LibraryComponent) GoToSongsFromPlaylistFilter(playlistId restApiV1.PlaylistId) {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeSongsFromPlaylist, playlistId: &playlistId})
}

func (c *LibraryComponent) GoToFavoritePlaylistsFromUserFilter(userId restApiV1.UserId) {
	c.historizeLibraryFilter(&libraryFilter{libraryType: libraryTypeFavoritePlaylistsFromUser, userId: &userId})
}

func (c *LibraryComponent) RefreshView() {
	// Redirection to menu when filter references unknown artist/album/playlist id
	currentFilter := c.currentFilter()

	if currentFilter.albumId != nil {
		if _, ok := c.uiApp.LocalDb().Albums[*currentFilter.albumId]; !ok {
			c.ResetToMenuFilter()
		}
	}
	if currentFilter.artistId != nil {
		if _, ok := c.uiApp.LocalDb().Artists[*currentFilter.artistId]; !ok {
			c.ResetToMenuFilter()
		}
	}
	if currentFilter.playlistId != nil {
		if _, ok := c.uiApp.LocalDb().Playlists[*currentFilter.playlistId]; !ok {
			c.ResetToMenuFilter()
		}
	}
	if currentFilter.userId != nil {
		if _, ok := c.uiApp.LocalDb().Users[*currentFilter.userId]; !ok {
			c.ResetToMenuFilter()
		}
	}

	currentFilter = c.currentFilter()
	c.list.Clear()
	oldIndex := currentFilter.index
	c.songs = nil
	title := c.getTitlePrefix() + ": " + currentFilter.libraryType.label()
	switch currentFilter.libraryType {
	case libraryTypeMenu:
		for _, libraryMenu := range libraryMenus {
			c.list.AddItem(libraryMenu.label())
		}
	case libraryTypeAllAlbums:
		for _, album := range c.uiApp.LocalDb().OrderedAlbums {
			c.list.AddItem(c.getMainTextAlbum(album, -1))
		}
	case libraryTypeAllArtists:
		for _, artist := range c.uiApp.LocalDb().OrderedArtists {
			c.list.AddItem(c.getMainTextArtist(artist, -1))
		}
	case libraryTypeAllPlaylists:
		c.playlists = c.uiApp.LocalDb().OrderedPlaylists
		c.loadPlaylists(c.playlists, nil)
	case libraryTypeAllSongs:
		c.songs = c.uiApp.LocalDb().OrderedSongs
		c.loadSongs(c.songs, nil, nil)
	case libraryTypeAllUsers:
		for _, user := range c.uiApp.LocalDb().OrderedUsers {
			c.list.AddItem(c.getMainTextUser(user))
		}
	case libraryTypeSongsFromAlbum:
		album := c.uiApp.LocalDb().Albums[*currentFilter.albumId]
		title = fmt.Sprintf(title, album.Name)
		c.songs = c.uiApp.LocalDb().AlbumOrderedSongs[album.Id]
		c.loadSongs(c.songs, currentFilter.albumId, nil)
	case libraryTypeSongsFromUnknownAlbum:
		c.songs = c.uiApp.LocalDb().UnknownAlbumSongs
		c.loadSongs(c.songs, nil, nil)
	case libraryTypeSongsFromArtist:
		artist := c.uiApp.LocalDb().Artists[*currentFilter.artistId]
		title = fmt.Sprintf(title, artist.Name)
		c.songs = c.uiApp.LocalDb().ArtistOrderedSongs[artist.Id]
		c.loadSongs(c.songs, nil, currentFilter.artistId)
	case libraryTypeSongsFromUnknownArtist:
		c.songs = c.uiApp.LocalDb().UnknownArtistSongs
		c.loadSongs(c.songs, nil, nil)
	case libraryTypeSongsFromPlaylist:
		playlist := c.uiApp.LocalDb().Playlists[*currentFilter.playlistId]
		title = fmt.Sprintf(title, playlist.Name)
		songIds := c.uiApp.LocalDb().Playlists[playlist.Id].SongIds
		for _, songId := range songIds {
			c.songs = append(c.songs, c.uiApp.LocalDb().Songs[songId])
		}
		c.loadSongs(c.songs, nil, nil)
	case libraryTypeFavoritePlaylistsFromUser:
		user := c.uiApp.LocalDb().Users[*currentFilter.userId]
		title = fmt.Sprintf(title, user.Name)
		c.playlists = c.uiApp.LocalDb().UserOrderedFavoritePlaylists[*currentFilter.userId]
		c.loadPlaylists(c.playlists, nil)
	}
	c.title.SetText(title)
	c.list.SetCurrentItem(oldIndex)
}

func (c *LibraryComponent) loadSongs(songs []*restApiV1.Song, fromAlbumId *restApiV1.AlbumId, fromArtistId *restApiV1.ArtistId) {
	for _, song := range songs {
		c.list.AddItem(c.getMainTextSong(song, fromAlbumId, fromArtistId, -1))
	}
}

func (c *LibraryComponent) getPositionnedIdSong(song *restApiV1.Song, fromAlbumId *restApiV1.AlbumId, fromArtistId *restApiV1.ArtistId, highlightPosition int) (songId *restApiV1.SongId, artistId *restApiV1.ArtistId, albumId *restApiV1.AlbumId) {
	currentPosition := 0

	// Song name
	if currentPosition == highlightPosition {
		return &song.Id, nil, nil
	}
	currentPosition++

	// Album name
	if song.AlbumId != "" && fromAlbumId == nil {
		if currentPosition == highlightPosition {
			return nil, nil, &song.AlbumId
		}
		currentPosition++
	}

	if len(song.ArtistIds) > 0 {
		for _, artistId := range song.ArtistIds {
			if fromArtistId == nil || (fromArtistId != nil && artistId != *fromArtistId) {
				if currentPosition == highlightPosition {
					return nil, &artistId, nil
				}
				currentPosition++
			}
		}
	}

	return nil, nil, nil
}

func (c *LibraryComponent) getMainTextSong(song *restApiV1.Song, fromAlbumId *restApiV1.AlbumId, fromArtistId *restApiV1.ArtistId, highlightPosition int) string {

	currentPosition := 0
	text := ""

	// Song name
	if currentPosition >= highlightPosition {
		underline := ""
		if currentPosition == highlightPosition {
			underline = "u"
		}
		text += "[" + ColorSongStr + "::" + underline + "]" + tview.Escape(song.Name) + "[white::-]"
	}
	currentPosition++

	// Album name
	if song.AlbumId != "" && fromAlbumId == nil {
		if currentPosition >= highlightPosition {
			if currentPosition > highlightPosition {
				text += " [::b]/[::-] "
			}
			underline := ""
			if currentPosition == highlightPosition {
				underline = "u"
			}
			text += "[" + ColorAlbumStr + "::" + underline + "]" + tview.Escape(c.uiApp.LocalDb().Albums[song.AlbumId].Name) + "[white::-]"
		}
		currentPosition++
	}

	if len(song.ArtistIds) > 0 {
		for _, artistId := range song.ArtistIds {
			if fromArtistId == nil || (fromArtistId != nil && artistId != *fromArtistId) {
				if currentPosition >= highlightPosition {
					if currentPosition > highlightPosition {
						text += " [::b]/[::-] "
					}
					underline := ""
					if currentPosition == highlightPosition {
						underline = "u"
					}
					text += "[" + ColorArtistStr + "::" + underline + "]" + tview.Escape(c.uiApp.LocalDb().Artists[artistId].Name) + "[white::-]"
				}
				currentPosition++
			}
		}
	}

	return text
}

func (c *LibraryComponent) getPositionnedIdAlbum(album *restApiV1.Album, highlightPosition int) (songId *restApiV1.SongId, artistId *restApiV1.ArtistId, albumId *restApiV1.AlbumId) {
	currentPosition := 0

	if album == nil {
		if currentPosition >= highlightPosition {
			return nil, nil, nil
		}
		currentPosition++
	} else {
		if currentPosition >= highlightPosition {
			return nil, nil, &album.Id
		}
		currentPosition++

		if len(album.ArtistIds) > 0 {
			for _, artistId := range album.ArtistIds {
				if currentPosition >= highlightPosition {
					return nil, &artistId, nil
				}
				currentPosition++
			}
		}
	}

	return nil, nil, nil
}

func (c *LibraryComponent) getMainTextAlbum(album *restApiV1.Album, highlightPosition int) string {
	currentPosition := 0
	text := ""

	if album == nil {
		if currentPosition >= highlightPosition {
			text += "[white]" + tview.Escape("(Unknown album)") + "[white] (" + strconv.Itoa(len(c.uiApp.LocalDb().UnknownAlbumSongs)) + ")"
		}
		currentPosition++
	} else {
		if currentPosition >= highlightPosition {
			text += "[" + ColorAlbumStr + "]" + tview.Escape(album.Name) + "[white] (" + strconv.Itoa(len(c.uiApp.LocalDb().AlbumOrderedSongs[album.Id])) + ")"
		}
		currentPosition++

		if len(album.ArtistIds) > 0 {
			for _, artistId := range album.ArtistIds {
				if currentPosition >= highlightPosition {
					if currentPosition > highlightPosition {
						text += " [::b]/[::-] "
					}
					text += "[" + ColorArtistStr + "]" + tview.Escape(c.uiApp.LocalDb().Artists[artistId].Name) + "[white]"
				}
				currentPosition++
			}
		}
	}

	return text
}

func (c *LibraryComponent) getPositionnedIdArtist(artist *restApiV1.Artist, highlightPosition int) (songId *restApiV1.SongId, artistId *restApiV1.ArtistId, albumId *restApiV1.AlbumId) {
	currentPosition := 0

	if artist == nil {
		if currentPosition >= highlightPosition {
			return nil, nil, nil
		}
		currentPosition++
	} else {
		if currentPosition >= highlightPosition {
			return nil, &artist.Id, nil
		}
		currentPosition++
	}

	return nil, nil, nil
}

func (c *LibraryComponent) getMainTextArtist(artist *restApiV1.Artist, highlightPosition int) string {
	currentPosition := 0
	text := ""

	if artist == nil {
		if currentPosition >= highlightPosition {
			text += "[white]" + tview.Escape("(Unknown artist)") + "[white] (" + strconv.Itoa(len(c.uiApp.LocalDb().UnknownArtistSongs)) + ")"
		}
		currentPosition++
	} else {
		if currentPosition >= highlightPosition {
			text += "[" + ColorArtistStr + "]" + tview.Escape(artist.Name) + "[white] (" + strconv.Itoa(len(c.uiApp.LocalDb().ArtistOrderedSongs[artist.Id])) + ")"
		}
		currentPosition++
	}

	return text
}

func (c *LibraryComponent) loadPlaylists(playlists []*restApiV1.Playlist, fromOwnerUserId *restApiV1.UserId) {
	for _, playlist := range playlists {
		c.list.AddItem(c.getMainTextPlaylist(playlist, fromOwnerUserId, -1))
	}
}

func (c *LibraryComponent) getMainTextPlaylist(playlist *restApiV1.Playlist, fromOwnerUserId *restApiV1.UserId, highlightPosition int) string {
	currentPosition := 0
	text := ""

	myFavoritePlaylists := c.uiApp.LocalDb().UserFavoritePlaylists[c.uiApp.ConnectedUserId()]
	if _, ok := myFavoritePlaylists[playlist.Id]; ok {
		text += "■ "
		//text += "💙"
	} else {
		text += "  "
	}

	if currentPosition >= highlightPosition {
		text += "[" + ColorPlaylistStr + "]" + tview.Escape(playlist.Name) + "[white] (" + strconv.Itoa(len(c.uiApp.LocalDb().Playlists[playlist.Id].SongIds)) + ")"
	}
	currentPosition++

	if len(playlist.OwnerUserIds) > 0 {
		for _, userId := range playlist.OwnerUserIds {
			if fromOwnerUserId == nil || (fromOwnerUserId != nil && userId != *fromOwnerUserId) {
				if currentPosition >= highlightPosition {
					if currentPosition > highlightPosition {
						text += " [::b]/[::-] "
					}
					text += "[" + ColorUserStr + "]" + tview.Escape(c.uiApp.LocalDb().Users[userId].Name) + "[white]"
				}
				currentPosition++
			}
		}
	}

	return text
}

func (c *LibraryComponent) getMainTextUser(user *restApiV1.User) string {

	userName := "[" + ColorUserStr + "]" + tview.Escape(user.Name)

	return userName
}

func (c *LibraryComponent) Focus(delegate func(tview.Primitive)) {
	delegate(c.list)
}

func (c *LibraryComponent) open(songId *restApiV1.SongId, artistId *restApiV1.ArtistId, albumId *restApiV1.AlbumId) {
	if songId != nil {
		c.uiApp.Play(*songId)
	}
	if artistId != nil {
		c.GoToSongsFromArtistFilter(*artistId)
	}
	if albumId != nil {
		c.GoToSongsFromAlbumFilter(*albumId)
	}
}
