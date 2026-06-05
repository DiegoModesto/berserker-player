import Foundation

// Modelos espelhando openapi.yaml. Campos opcionais para robustez a evoluções.

struct TokenPair: Codable {
    let accessToken: String
    let refreshToken: String?
    let expiresAt: String?
}

struct MediaTokenResponse: Codable {
    let token: String
    let expiresAt: String
}

struct User: Codable, Identifiable {
    let id: String
    let username: String
    let isAdmin: Bool
}

struct Artist: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    var albumCount: Int?
    var songCount: Int?
    var starred: Bool?
    var coverArtId: String?
}

struct Album: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    var artistId: String?
    var artistName: String?
    var year: Int?
    var genre: String?
    var songCount: Int?
    var duration: Int?
    var coverArtId: String?
    var starred: Bool?
    var playCount: Int?
}

struct Song: Codable, Identifiable, Hashable {
    let id: String
    let title: String
    var albumId: String?
    var albumName: String?
    var artistId: String?
    var artistName: String?
    var track: Int?
    var disc: Int?
    var duration: Int?
    var bitRate: Int?
    var suffix: String?
    var coverArtId: String?
    var starred: Bool?
    var rating: Int?
    var playCount: Int?
}

struct Playlist: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    var songCount: Int?
    var duration: Int?
}

struct Page<T: Codable>: Codable {
    let items: [T]
    let total: Int
    let offset: Int
    let limit: Int
}

struct AlbumDetail: Codable {
    let id: String
    let name: String
    var artistName: String?
    var year: Int?
    var coverArtId: String?
    var songs: [Song]
}

struct ArtistDetail: Codable {
    let id: String
    let name: String
    var albums: [Album]
}

struct PlaylistDetail: Codable {
    let id: String
    let name: String
    var songs: [Song]
}

struct SearchResult: Codable {
    var artists: [Artist]
    var albums: [Album]
    var songs: [Song]
}

struct ItemRef: Codable {
    let id: String
    let type: String
}

struct APIProblem: Codable, Error {
    let title: String
    let status: Int
    var detail: String?
}
