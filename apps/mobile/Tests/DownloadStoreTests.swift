import XCTest
import SwiftData
@testable import BerserkerPlayer

@MainActor
final class DownloadStoreTests: XCTestCase {
    private func makeSong(_ id: String) -> Song {
        let json = "{\"id\":\"\(id)\",\"title\":\"T-\(id)\",\"artistName\":\"Art\",\"albumName\":\"Alb\",\"suffix\":\"mp3\"}"
            .data(using: .utf8)!
        return try! JSONDecoder().decode(Song.self, from: json)
    }

    private func makeStore() throws -> DownloadStore {
        let config = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(for: DownloadedTrack.self, configurations: config)
        let dir = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent("dl-test-" + UUID().uuidString)
        return DownloadStore(context: ModelContext(container), directory: dir)
    }

    private func tempFile(bytes: Int) throws -> URL {
        let url = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent(UUID().uuidString + ".mp3")
        try Data(repeating: 0x42, count: bytes).write(to: url)
        return url
    }

    func testRecordAndQuery() throws {
        let store = try makeStore()
        let song = makeSong("s1")
        XCTAssertFalse(store.isDownloaded("s1"))

        let src = try tempFile(bytes: 1024)
        try store.record(song: song, sourceFile: src)

        XCTAssertTrue(store.isDownloaded("s1"))
        XCTAssertNotNil(store.localURL("s1"))
        XCTAssertEqual(store.totalBytes(), 1024)
        XCTAssertEqual(store.all().count, 1)
        XCTAssertEqual(store.track(for: "s1")?.title, "T-s1")
        // O arquivo de origem foi movido (não existe mais).
        XCTAssertFalse(FileManager.default.fileExists(atPath: src.path))
    }

    func testTotalBytesMultiple() throws {
        let store = try makeStore()
        try store.record(song: makeSong("s1"), sourceFile: try tempFile(bytes: 1000))
        try store.record(song: makeSong("s2"), sourceFile: try tempFile(bytes: 500))
        XCTAssertEqual(store.totalBytes(), 1500)
        XCTAssertEqual(store.all().count, 2)
    }

    func testRemove() throws {
        let store = try makeStore()
        try store.record(song: makeSong("s1"), sourceFile: try tempFile(bytes: 10))
        let url = store.localURL("s1")!
        try store.remove("s1")
        XCTAssertFalse(store.isDownloaded("s1"))
        XCTAssertNil(store.localURL("s1"))
        XCTAssertFalse(FileManager.default.fileExists(atPath: url.path))
    }

    func testClearAll() throws {
        let store = try makeStore()
        try store.record(song: makeSong("s1"), sourceFile: try tempFile(bytes: 10))
        try store.record(song: makeSong("s2"), sourceFile: try tempFile(bytes: 20))
        try store.clearAll()
        XCTAssertEqual(store.all().count, 0)
        XCTAssertEqual(store.totalBytes(), 0)
    }

    func testReRecordUpdatesNotDuplicates() throws {
        let store = try makeStore()
        try store.record(song: makeSong("s1"), sourceFile: try tempFile(bytes: 10))
        try store.record(song: makeSong("s1"), sourceFile: try tempFile(bytes: 30))
        XCTAssertEqual(store.all().count, 1)
        XCTAssertEqual(store.totalBytes(), 30)
    }
}
