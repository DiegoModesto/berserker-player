import XCTest
@testable import BerserkerPlayer

private func songs(_ ids: [String]) -> [Song] {
    ids.map { id in
        let json = "{\"id\":\"\(id)\",\"title\":\"\(id)\"}".data(using: .utf8)!
        return try! JSONDecoder().decode(Song.self, from: json)
    }
}

final class PlaybackQueueTests: XCTestCase {
    func testNextAdvances() {
        let q = PlaybackQueue(songs: songs(["a", "b", "c"]), index: 0)
        XCTAssertEqual(q.nextIndex(), 1)
    }

    func testNextAtEndWithoutRepeatStops() {
        let q = PlaybackQueue(songs: songs(["a", "b"]), index: 1)
        XCTAssertNil(q.nextIndex())
    }

    func testRepeatAllWraps() {
        let q = PlaybackQueue(songs: songs(["a", "b"]), index: 1, repeatMode: .all)
        XCTAssertEqual(q.nextIndex(), 0)
    }

    func testRepeatOneStaysSame() {
        let q = PlaybackQueue(songs: songs(["a", "b"]), index: 1, repeatMode: .one)
        XCTAssertEqual(q.nextIndex(), 1)
    }

    func testShuffleUsesInjectedRandom() {
        let q = PlaybackQueue(songs: songs(["a", "b", "c"]), index: 0, shuffle: true)
        XCTAssertEqual(q.nextIndex(randomIndex: { _ in 2 }), 2)
    }

    func testEmptyQueueNextIsNil() {
        let q = PlaybackQueue()
        XCTAssertNil(q.nextIndex())
        XCTAssertTrue(q.isEmpty)
        XCTAssertNil(q.current)
    }

    func testPreviousRestartsWhenPastThreshold() {
        let q = PlaybackQueue(songs: songs(["a", "b", "c"]), index: 2)
        XCTAssertEqual(q.previousIndex(currentTime: 5), 2) // reinicia
        XCTAssertEqual(q.previousIndex(currentTime: 1), 1) // volta
    }

    func testPreviousClampsAtZero() {
        let q = PlaybackQueue(songs: songs(["a"]), index: 0)
        XCTAssertEqual(q.previousIndex(currentTime: 0), 0)
    }

    func testCurrentReturnsSong() {
        let q = PlaybackQueue(songs: songs(["a", "b"]), index: 1)
        XCTAssertEqual(q.current?.id, "b")
    }
}
