#!/usr/bin/env python3
"""Create a demo fence, vehicle and rule, then move outside → inside → outside."""
import datetime, json, os, time, urllib.request
BASE = os.environ.get('API_URL', 'http://localhost:8080').rstrip('/')
def post(path, body):
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode(), headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req) as response:
        result = json.load(response)
    print(path, json.dumps(result))
    return result
stamp = str(int(time.time()))
fence = post('/geofences', {'name': 'Demo depot ' + stamp, 'description': 'San Francisco demo', 'category': 'restricted_zone', 'coordinates': [[37.77,-122.43],[37.785,-122.43],[37.785,-122.41],[37.77,-122.41],[37.77,-122.43]]})
vehicle = post('/vehicles', {'vehicle_number': 'DEMO-' + stamp, 'driver_name': 'Demo Driver', 'vehicle_type': 'truck', 'phone': '+10000000000'})
post('/alerts/configure', {'geofence_id': fence['id'], 'vehicle_id': vehicle['id'], 'event_type': 'both'})
print('Open the dashboard in two tabs to watch the entry and exit alerts.')
for lat, lng in [(37.76,-122.44),(37.777,-122.42),(37.76,-122.44)]:
    time.sleep(2)
    post('/vehicles/location', {'vehicle_id': vehicle['id'], 'latitude': lat, 'longitude': lng, 'timestamp': datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00','Z')})
