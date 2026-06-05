import Foundation
import Observation

/// Coordena downloads de faixas para uso offline (rede + persistência).
@MainActor
@Observable
final class DownloadManager {
    private let api: APIClient
    let store: DownloadStore
    private(set) var inProgress: Set<String> = []

    init(api: APIClient, store: DownloadStore) {
        self.api = api
        self.store = store
    }

    func isDownloaded(_ songID: String) -> Bool { store.isDownloaded(songID) }
    func localURL(_ songID: String) -> URL? { store.localURL(songID) }

    /// Baixa a faixa (formato original) e registra para reprodução offline.
    func download(_ song: Song) async {
        guard !store.isDownloaded(song.id), !inProgress.contains(song.id) else { return }
        inProgress.insert(song.id)
        defer { inProgress.remove(song.id) }
        do {
            let url = try await api.streamURL(songId: song.id) // raw (sem transcode)
            let (tmp, _) = try await URLSession.shared.download(from: url)
            try store.record(song: song, sourceFile: tmp)
        } catch {
            // Falha de download é silenciosa na UI; o item simplesmente não fica offline.
        }
    }

    func remove(_ songID: String) {
        try? store.remove(songID)
    }

    func clearAll() {
        try? store.clearAll()
    }
}
