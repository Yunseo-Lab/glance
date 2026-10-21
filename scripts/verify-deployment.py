#!/usr/bin/env python3
"""Read-only acceptance and bounded load check against your own deployment."""
import argparse
import base64
import concurrent.futures
import http.cookiejar
import json
import ssl
import statistics
import time
import urllib.parse
import urllib.request

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--url', required=True)
parser.add_argument('--password-file', default='.secrets/owner_password')
parser.add_argument('--ca', help='Optional trusted CA certificate for a private deployment')
args = parser.parse_args()
base = args.url.rstrip('/')
url = urllib.parse.urlsplit(base)
if url.username or url.password or url.path or url.query or url.fragment:
    parser.error('URL must be an origin without credentials')
if url.scheme != 'https' and not (url.scheme == 'http' and url.hostname in ('localhost', '127.0.0.1', '::1')):
    parser.error('Use HTTPS, or HTTP on loopback only')
with open(args.password_file, encoding='utf-8') as stream:
    password = stream.read().strip()
context = ssl.create_default_context(cafile=args.ca)
cookies = http.cookiejar.CookieJar()
class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *unused):
        return None
def proxy_handler():
    return urllib.request.ProxyHandler({}) if url.hostname in ('localhost', '127.0.0.1', '::1') else urllib.request.ProxyHandler()
opener = urllib.request.build_opener(proxy_handler(), urllib.request.HTTPSHandler(context=context), urllib.request.HTTPCookieProcessor(cookies), NoRedirect())
with opener.open(base + '/', timeout=10) as response:
    assert response.status == 200
    if url.scheme == 'https':
        assert response.headers.get('Strict-Transport-Security')
request = urllib.request.Request(base + '/api/v1/auth/login', data=json.dumps({'password': password}).encode(), headers={'Content-Type': 'application/json', 'Origin': base})
with opener.open(request, timeout=10) as response:
    assert response.status == 200
assert list(cookies), 'Login did not create a session'
if url.scheme == 'https':
    assert all(cookie.secure for cookie in cookies), 'Session must be Secure'
with opener.open(base + '/api/readyz', timeout=10) as response:
    assert json.load(response)['status'] == 'ready'
authorization = 'Basic ' + base64.b64encode(('owner:' + password).encode()).decode()
def read_board(_):
    client = urllib.request.build_opener(proxy_handler(), urllib.request.HTTPSHandler(context=context), NoRedirect())
    request = urllib.request.Request(base + '/api/v1/dashboard', headers={'Authorization': authorization})
    start = time.monotonic()
    with client.open(request, timeout=10) as response:
        assert response.status == 200
        assert json.load(response)['id'] == 'home'
    return (time.monotonic() - start) * 1000
with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
    timings = sorted(pool.map(read_board, range(200)))
print(f'PASS: TLS/headers (when HTTPS), login, session, readiness; 200/200 reads, 8 workers, median={statistics.median(timings):.1f}ms, p95={timings[189]:.1f}ms')
