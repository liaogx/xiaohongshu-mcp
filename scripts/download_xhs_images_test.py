"""Offline tests with synthetic API responses; no account or remote image access."""
import io
import tempfile
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch
from urllib.error import HTTPError
from urllib.request import Request

import download_xhs_images as subject


class DownloadTests(unittest.TestCase):
    def test_names_and_urls(self):
        self.assertNotIn('/', subject.clean_name('../example/name', 'fallback'))
        self.assertEqual(subject.clean_name('', 'fallback'), 'fallback')
        self.assertEqual(subject.unique_urls(['https://example.com/a', None, 'https://example.com/a']),
                         ['https://example.com/a'])
        self.assertEqual(subject.cover_urls({'noteCard': {'cover': {'urlDefault': 'example-cover'}}}),
                         ['example-cover'])
        self.assertEqual(subject.detail_image_urls({'data': {'data': {'note': {
            'imageList': [{'urlDefault': 'example-image'}]}}}}), ['example-image'])

    @patch.dict('os.environ', {'AUTH_TOKEN': 'example-only'})
    def test_api_auth_and_error_status(self):
        opener = MagicMock()
        opener.open.return_value.__enter__.return_value.read.return_value = b'{"success":true,"data":{}}'
        with patch.object(subject, 'build_opener', return_value=opener):
            self.assertTrue(subject.json_request('http://127.0.0.1:18060/health', timeout=1)['success'])
            self.assertEqual(opener.open.call_args.args[0].get_header('Authorization'), 'Bearer example-only')
            opener.open.return_value.__enter__.return_value.read.return_value = b'{"success":false}'
            with self.assertRaises(RuntimeError):
                subject.json_request('http://127.0.0.1:18060/health', timeout=1)

    def test_api_redirect_is_not_followed(self):
        with self.assertRaises(HTTPError):
            subject.NoAPIRedirect().redirect_request(
                Request('http://127.0.0.1:18060/health'), io.BytesIO(), 302, '', {}, 'https://example.com')

    @patch.dict('os.environ', {'AUTH_TOKEN': 'example-only'})
    def test_image_request_has_no_api_auth_and_gallery_escapes_text(self):
        response = MagicMock()
        response.__enter__.return_value.headers = {'Content-Type': 'image/png'}
        response.__enter__.return_value.read.return_value = b'synthetic-image'
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            with patch.object(subject, 'urlopen', return_value=response) as request:
                ok, path = subject.download_image('https://example.com/image.png', root / 'cover', 1)
                self.assertTrue(ok)
                self.assertIsNone(request.call_args.args[0].get_header('Authorization'))
                self.assertEqual(Path(path).read_bytes(), b'synthetic-image')
            subject.write_index(root, [{'rank': 1, 'title': '<script>example</script>',
                                        'author': '<example>', 'files': ['cover.png']}])
            page = (root / 'index.html').read_text()
            self.assertNotIn('<script>', page)
            self.assertIn('&lt;script&gt;', page)

    def test_error_does_not_disclose_url(self):
        error = HTTPError('https://example.com/private', 403, 'example', {}, None)
        self.assertEqual(subject.safe_error(error), 'HTTP 403')


if __name__ == '__main__':
    unittest.main()
