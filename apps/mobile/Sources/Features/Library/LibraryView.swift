import SwiftUI

@MainActor
@Observable
final class LibraryViewModel {
    var albums: [Album] = []
    var loading = false
    var filter = "all"
    private var offset = 0
    private var total = 0
    private let api: APIClient

    init(api: APIClient) { self.api = api }

    func reload() async {
        offset = 0; albums = []
        await loadMore()
    }

    func loadMore() async {
        guard !loading, offset == 0 || offset < total else { return }
        loading = true
        defer { loading = false }
        do {
            let page = try await api.albums(filter: filter, offset: offset, limit: 40)
            albums.append(contentsOf: page.items)
            total = page.total
            offset += page.items.count
        } catch {}
    }
}

struct LibraryView: View {
    @Environment(Session.self) private var session
    @Environment(PlaybackEngine.self) private var playback
    @State private var model: LibraryViewModel?

    private let columns = [GridItem(.adaptive(minimum: 150), spacing: 16)]

    var body: some View {
        NavigationStack {
            ScrollView {
                Picker("Filtro", selection: Binding(
                    get: { model?.filter ?? "all" },
                    set: { newValue in
                        guard let m = model else { return }
                        m.filter = newValue
                        Task { await m.reload() }
                    })) {
                    Text("Todos").tag("all")
                    Text("Recentes").tag("recent")
                    Text("Mais tocados").tag("frequent")
                    Text("Aleatório").tag("random")
                }
                .pickerStyle(.segmented).padding(.horizontal)

                LazyVGrid(columns: columns, spacing: 16) {
                    ForEach(model?.albums ?? []) { album in
                        NavigationLink(value: album) {
                            AlbumCard(album: album)
                        }
                        .buttonStyle(.plain)
                        .task {
                            if album.id == model?.albums.last?.id { await model?.loadMore() }
                        }
                    }
                }
                .padding()
            }
            .navigationTitle("Álbuns")
            .navigationDestination(for: Album.self) { AlbumDetailView(albumID: $0.id) }
        }
        .task {
            if model == nil {
                let m = LibraryViewModel(api: session.api)
                model = m
                await m.reload()
            }
        }
    }
}

struct AlbumCard: View {
    let album: Album
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            CoverImage(coverArtId: album.coverArtId ?? album.id, size: 160)
            Text(album.name).font(.subheadline.weight(.medium)).lineLimit(1)
            Text(album.artistName ?? "").font(.caption).foregroundStyle(.secondary).lineLimit(1)
        }
    }
}
