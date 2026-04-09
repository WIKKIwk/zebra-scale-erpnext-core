import 'dart:async';
import 'dart:convert';
import 'dart:io';

class DiscoveryAnnouncement {
  const DiscoveryAnnouncement({
    required this.host,
    required this.httpPort,
    required this.serverName,
    required this.serverRef,
    required this.displayName,
    required this.role,
    required this.latencyMs,
  });

  final String host;
  final int httpPort;
  final String serverName;
  final String serverRef;
  final String displayName;
  final String role;
  final int latencyMs;
}

const _discoveryProbeV1 = 'GSCALE_DISCOVER_V1';

Future<List<String>> collectCandidateHosts() async {
  final hosts = <String>{'127.0.0.1', '10.0.2.2', 'gscale.local'};

  final interfaces = await NetworkInterface.list(
    includeLoopback: true,
    includeLinkLocal: false,
    type: InternetAddressType.IPv4,
  );

  for (final iface in interfaces) {
    for (final address in iface.addresses) {
      final host = address.address.trim();
      if (_isPrivateIPv4(host) || host == '127.0.0.1') {
        hosts.add(host);
      }
    }
  }

  final ipv4Hosts = hosts.where((host) => _looksLikeIPv4(host)).toList()
    ..sort(_compareIPv4);
  final namedHosts = hosts.where((host) => !_looksLikeIPv4(host)).toList()
    ..sort();
  return [...ipv4Hosts, ...namedHosts];
}

Future<List<DiscoveryAnnouncement>> discoverAnnouncements({
  required int port,
  required Duration timeout,
}) async {
  final socket = await RawDatagramSocket.bind(
    InternetAddress.anyIPv4,
    0,
    reuseAddress: true,
  );
  socket.broadcastEnabled = true;

  final stopwatch = Stopwatch()..start();
  final results = <String, DiscoveryAnnouncement>{};
  final done = Completer<void>();
  late final StreamSubscription<RawSocketEvent> sub;

  sub = socket.listen((event) {
    if (event != RawSocketEvent.read) {
      return;
    }
    final datagram = socket.receive();
    if (datagram == null) {
      return;
    }
    final payload = jsonDecode(utf8.decode(datagram.data));
    if (payload is! Map<String, dynamic>) {
      return;
    }
    if ((payload['service']?.toString().trim() ?? '') != 'mobileapi') {
      return;
    }

    final host = datagram.address.address.trim();
    final announcement = DiscoveryAnnouncement(
      host: host,
      httpPort: _asInt(payload['http_port']) ?? 8081,
      serverName: payload['server_name']?.toString().trim() ?? host,
      serverRef: payload['server_ref']?.toString().trim() ?? '',
      displayName: payload['display_name']?.toString().trim() ?? 'Operator',
      role: payload['role']?.toString().trim() ?? 'operator',
      latencyMs: stopwatch.elapsedMilliseconds,
    );
    final key = '${announcement.serverRef}|${announcement.serverName}|$host';
    results[key] = announcement;
  });

  final targets = await _collectBroadcastTargets();
  final packet = utf8.encode(_discoveryProbeV1);
  for (final target in targets) {
    socket.send(packet, target, port);
  }

  Timer(timeout, () {
    if (!done.isCompleted) {
      done.complete();
    }
  });

  await done.future;
  await sub.cancel();
  socket.close();
  return results.values.toList();
}

Future<List<InternetAddress>> _collectBroadcastTargets() async {
  final out = <InternetAddress>{InternetAddress('255.255.255.255')};
  final interfaces = await NetworkInterface.list(
    includeLoopback: false,
    includeLinkLocal: false,
    type: InternetAddressType.IPv4,
  );

  for (final iface in interfaces) {
    for (final address in iface.addresses) {
      final host = address.address.trim();
      if (!_isPrivateIPv4(host)) {
        continue;
      }
      final parts = host.split('.');
      if (parts.length != 4) {
        continue;
      }
      out.add(InternetAddress('${parts[0]}.${parts[1]}.${parts[2]}.255'));
    }
  }

  return out.toList();
}

int? _asInt(Object? value) {
  if (value is int) {
    return value;
  }
  return int.tryParse(value?.toString() ?? '');
}

bool _isPrivateIPv4(String host) {
  final parts = host.split('.');
  if (parts.length != 4) {
    return false;
  }

  final values = parts.map(int.tryParse).toList();
  if (values.any((value) => value == null)) {
    return false;
  }

  final first = values[0]!;
  final second = values[1]!;
  if (first == 10) {
    return true;
  }
  if (first == 172 && second >= 16 && second <= 31) {
    return true;
  }
  if (first == 192 && second == 168) {
    return true;
  }
  return false;
}

bool _looksLikeIPv4(String host) {
  return host.split('.').length == 4;
}

int _compareIPv4(String left, String right) {
  final leftParts = left.split('.').map(int.parse).toList();
  final rightParts = right.split('.').map(int.parse).toList();
  for (var i = 0; i < 4; i++) {
    final cmp = leftParts[i].compareTo(rightParts[i]);
    if (cmp != 0) {
      return cmp;
    }
  }
  return 0;
}
