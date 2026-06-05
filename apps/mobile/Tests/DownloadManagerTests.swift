import XCTest
import SwiftData
@testable import BerserkerPlayer

@MainActor
final class DownloadManagerTests: XCTestCase {
    private func makeManager() throws -> (DownloadManager, DownloadStore) {
        let config = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(for: DownloadedTrack.self, configurations: config)
        let dir = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent("mgr-test-" + UUID().uuidString)
        let store = DownloadStore(context: ModelContext(container), directory: dir)
        let api = APIClient(baseURL: URL(string: "http://localhost:4533")!)
        return (DownloadManager(api: api, store: store), store)
    }

    private func makeSong(_ id: String) -> Song {
        let json = "{\"id\":\"\(id)\",\"title\":\"T\",\"suffix\":\"mp3\"}".data(using: .utf8)!
        return try! JSONDecoder().decode(Song.self, from: json)
    }

    private func tempFile() throws -> URL {
        let url = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent(UUID().uuidString + ".mp3")
        try Data(repeating: 1, count: 64).write(to: url)
        return url
    }

    func testDelegatesToStore() throws {
        let (manager, store) = try makeManager()
        XCTAssertFalse(manager.isDownloaded("s1"))
        try store.record(song: makeSong("s1"), sourceFile: try tempFile())
        XCTAssertTrue(manager.isDownloaded("s1"))
        XCTAssertNotNil(manager.localURL("s1"))
    }

    func testRemoveAndClear() throws {
        let (manager, store) = try makeManager()
        try store.record(song: makeSong("s1"), sourceFile: try tempFile())
        try store.record(song: makeSong("s2"), sourceFile: try tempFile())
        manager.remove("s1")
        XCTAssertFalse(manager.isDownloaded("s1"))
        XCTAssertTrue(manager.isDownloaded("s2"))
        manager.clearAll()
        XCTAssertEqual(store.all().count, 0)
    }

    func testDownloadSkipsWhenAlreadyPresent() async throws {
        let (manager, store) = try makeManager()
        try store.record(song: makeSong("s1"), sourceFile: try tempFile())
        // Já baixado: download() retorna sem tocar a rede (guard).
        await manager.download(makeSong("s1"))
        XCTAssertEqual(store.all().count, 1)
        XCTAssertTrue(manager.inProgress.isEmpty)
    }
}
