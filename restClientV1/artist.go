package restClientV1

import (
	"bytes"
	"encoding/json"
	"github.com/jypelle/mifasol/restApiV1"
)

func (c *RestClient) CreateArtist(artistMeta *restApiV1.ArtistMeta) (*restApiV1.Artist, ClientError) {
	var artist *restApiV1.Artist

	encodedArtistMeta, _ := json.Marshal(artistMeta)

	response, cliErr := c.doPostRequest("/artists", JsonContentType, bytes.NewBuffer(encodedArtistMeta))
	if cliErr != nil {
		return nil, cliErr
	}
	defer response.Body.Close()

	err := json.NewDecoder(response.Body).Decode(&artist)
	if err != nil {
		return nil, NewClientError(err)
	}

	return artist, nil
}

func (c *RestClient) UpdateArtist(artistId string, artistMeta *restApiV1.ArtistMeta) (*restApiV1.Artist, ClientError) {
	var artist *restApiV1.Artist

	encodedArtistMeta, _ := json.Marshal(artistMeta)

	response, cliErr := c.doPutRequest("/artists/"+artistId, JsonContentType, bytes.NewBuffer(encodedArtistMeta))
	if cliErr != nil {
		return nil, cliErr
	}
	defer response.Body.Close()

	err := json.NewDecoder(response.Body).Decode(&artist)
	if err != nil {
		return nil, NewClientError(err)
	}

	return artist, nil
}

func (c *RestClient) DeleteArtist(artistId string) (*restApiV1.Artist, ClientError) {
	var artist *restApiV1.Artist

	response, cliErr := c.doDeleteRequest("/artists/" + artistId)
	if cliErr != nil {
		return nil, cliErr
	}
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(&artist); err != nil {
		return nil, NewClientError(err)
	}

	return artist, nil
}
