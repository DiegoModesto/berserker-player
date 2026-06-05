import XCTest
@testable import BerserkerPlayer

final class APIModelsTests: XCTestCase {
    func testDecodeSong() throws {
        let json = """
        {"id":"abc","title":"Guts","albumId":"al1","albumName":"OST",
         "artistName":"Hirasawa","duration":214,"suffix":"mp3","starred":true,"rating":4}
        """.data(using: .utf8)!
        let song = try JSONDecoder().decode(Song.self, from: json)
        XCTAssertEqual(song.id, "abc")
        XCTAssertEqual(song.title, "Guts")
        XCTAssertEqual(song.duration, 214)
        XCTAssertEqual(song.starred, true)
        XCTAssertEqual(song.rating, 4)
    }

    func testDecodePage() throws {
        let json = """
        {"items":[{"id":"1","name":"A"}],"total":1,"offset":0,"limit":50}
        """.data(using: .utf8)!
        let page = try JSONDecoder().decode(Page<Album>.self, from: json)
        XCTAssertEqual(page.total, 1)
        XCTAssertEqual(page.items.first?.name, "A")
    }

    func testTimeStringFormatting() {
        XCTAssertEqual(214.asTimeString, "3:34")
        XCTAssertEqual(5.asTimeString, "0:05")
    }
}
