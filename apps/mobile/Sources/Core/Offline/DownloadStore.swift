import Foundation
import SwiftData

/// Gerencia metadados (SwiftData) e arquivos de faixas baixadas.
@MainActor
final class DownloadStore {
    private let context: ModelContext
    let directory: URL

    init(context: ModelContext, directory: URL) {
        self.context = context
        self.directory = directory
        try? FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    }

    func isDownloaded(_ songID: String) -> Bool {
        track(for: songID) != nil
    }

    /// URL local do arquivo, se baixado e presente em disco.
    func localURL(_ songID: String) -> URL? {
        guard let t = track(for: songID) else { return nil }
        let url = directory.appendingPathComponent(t.fileName)
        return FileManager.default.fileExists(atPath: url.path) ? url : nil
    }

    func track(for songID: String) -> DownloadedTrack? {
        // Filtro em Swift (evita o macro #Predicate); o nº de downloads é pequeno.
        all().first { $0.songID == songID }
    }

    func all() -> [DownloadedTrack] {
        let fetched = (try? context.fetch(FetchDescriptor<DownloadedTrack>())) ?? []
        return fetched.sorted { $0.downloadedAt > $1.downloadedAt }
    }

    func totalBytes() -> Int {
        all().reduce(0) { $0 + $1.byteSize }
    }

    /// Move `sourceFile` para o diretório de downloads e registra os metadados.
    @discardableResult
    func record(song: Song, sourceFile: URL) throws -> DownloadedTrack {
        let fileName = song.id + "." + (song.suffix ?? "audio")
        let dest = directory.appendingPathComponent(fileName)
        let fm = FileManager.default
        if fm.fileExists(atPath: dest.path) { try? fm.removeItem(at: dest) }
        try fm.moveItem(at: sourceFile, to: dest)
        let size = ((try? fm.attributesOfItem(atPath: dest.path))?[.size] as? Int) ?? 0

        if let existing = track(for: song.id) {
            existing.fileName = fileName
            existing.byteSize = size
            existing.downloadedAt = Date()
            try context.save()
            return existing
        }
        let t = DownloadedTrack(
            songID: song.id, title: song.title,
            artistName: song.artistName ?? "", albumName: song.albumName ?? "",
            fileName: fileName, byteSize: size)
        context.insert(t)
        try context.save()
        return t
    }

    func remove(_ songID: String) throws {
        guard let t = track(for: songID) else { return }
        let url = directory.appendingPathComponent(t.fileName)
        try? FileManager.default.removeItem(at: url)
        context.delete(t)
        try context.save()
    }

    func clearAll() throws {
        for t in all() { try remove(t.songID) }
    }
}
