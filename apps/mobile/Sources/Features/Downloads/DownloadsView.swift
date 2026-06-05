import SwiftUI

struct DownloadsView: View {
    @Environment(DownloadManager.self) private var downloads
    @State private var tracks: [DownloadedTrack] = []
    @State private var totalBytes = 0

    var body: some View {
        List {
            if tracks.isEmpty {
                ContentUnavailableView("Nada baixado", systemImage: "arrow.down.circle",
                                       description: Text("Baixe faixas para ouvir offline."))
            } else {
                Section {
                    ForEach(tracks, id: \.songID) { t in
                        VStack(alignment: .leading) {
                            Text(t.title).lineLimit(1)
                            Text(t.artistName).font(.caption).foregroundStyle(.secondary)
                        }
                    }
                    .onDelete(perform: deleteAt)
                } footer: {
                    Text("Armazenamento: \(formatted(totalBytes))")
                }
                Section {
                    Button(role: .destructive) {
                        downloads.clearAll()
                        refresh()
                    } label: {
                        Text("Limpar todos os downloads")
                    }
                }
            }
        }
        .navigationTitle("Downloads")
        .task { refresh() }
    }

    private func refresh() {
        tracks = downloads.store.all()
        totalBytes = downloads.store.totalBytes()
    }

    private func deleteAt(_ offsets: IndexSet) {
        for i in offsets { downloads.remove(tracks[i].songID) }
        refresh()
    }

    private func formatted(_ bytes: Int) -> String {
        ByteCountFormatter.string(fromByteCount: Int64(bytes), countStyle: .file)
    }
}
