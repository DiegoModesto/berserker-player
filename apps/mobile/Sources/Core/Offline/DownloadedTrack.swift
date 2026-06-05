import Foundation
import SwiftData

/// Faixa baixada para reprodução offline (metadados; o áudio fica em arquivo).
@Model
final class DownloadedTrack {
    @Attribute(.unique) var songID: String
    var title: String
    var artistName: String
    var albumName: String
    var fileName: String
    var byteSize: Int
    var downloadedAt: Date

    init(songID: String, title: String, artistName: String, albumName: String,
         fileName: String, byteSize: Int, downloadedAt: Date = Date()) {
        self.songID = songID
        self.title = title
        self.artistName = artistName
        self.albumName = albumName
        self.fileName = fileName
        self.byteSize = byteSize
        self.downloadedAt = downloadedAt
    }
}
