import 'dart:async';
import 'dart:convert';

import 'package:device_preview/device_preview.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'network_candidates_stub.dart'
    if (dart.library.io) 'network_candidates_io.dart'
    as network_candidates;

const _defaultApiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://127.0.0.1:8081',
);
const _defaultApiPort = 8081;
const _discoveryPort = 18081;
const _fastProbeTimeout = Duration(milliseconds: 180);
const _manualProbeTimeout = Duration(seconds: 2);
const _udpDiscoveryTimeout = Duration(milliseconds: 450);
const _lastServerKey = 'last_server_base_url';
const _m3Surface = Color(0xFFF4EEFF);
const _m3Container = Color(0xFFDCD6F7);
const _m3Accent = Color(0xFFA6B1E1);
const _m3Primary = Color(0xFF424874);

bool get previewEnabled {
  if (kReleaseMode) {
    return false;
  }
  if (kIsWeb) {
    return true;
  }

  switch (defaultTargetPlatform) {
    case TargetPlatform.android:
    case TargetPlatform.iOS:
      return false;
    case TargetPlatform.linux:
    case TargetPlatform.macOS:
    case TargetPlatform.windows:
    case TargetPlatform.fuchsia:
      return true;
  }
}

void main() {
  runApp(
    DevicePreview(
      enabled: previewEnabled,
      isToolbarVisible: true,
      tools: const [...DevicePreview.defaultTools],
      builder: (context) => const GScaleMobileApp(),
    ),
  );
}

class GScaleMobileApp extends StatefulWidget {
  const GScaleMobileApp({super.key});

  @override
  State<GScaleMobileApp> createState() => _GScaleMobileAppState();
}

class _GScaleMobileAppState extends State<GScaleMobileApp> {
  DiscoveredServer? _selectedServer;

  Future<void> _openServer(DiscoveredServer server) async {
    await saveLastUsedServer(server.endpoint);
    if (!mounted) {
      return;
    }
    setState(() {
      _selectedServer = server;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'GScale Mobile',
      debugShowCheckedModeBanner: false,
      locale: previewEnabled ? DevicePreview.locale(context) : null,
      builder: previewEnabled ? DevicePreview.appBuilder : null,
      themeMode: ThemeMode.system,
      theme: buildAppTheme(Brightness.light),
      darkTheme: buildAppTheme(Brightness.dark),
      home: _selectedServer == null
          ? ServerPickerPage(onOpenServer: _openServer)
          : OperatorDashboardPage(
              server: _selectedServer!,
              onChangeServer: () {
                setState(() {
                  _selectedServer = null;
                });
              },
            ),
    );
  }
}

ThemeData buildAppTheme(Brightness brightness) {
  final isDark = brightness == Brightness.dark;
  final seedScheme = ColorScheme.fromSeed(
    seedColor: _m3Primary,
    brightness: brightness,
  );
  final scheme = isDark
      ? seedScheme
      : seedScheme.copyWith(
          primary: _m3Primary,
          onPrimary: _m3Surface,
          primaryContainer: _m3Container,
          onPrimaryContainer: _m3Primary,
          secondary: _m3Accent,
          onSecondary: _m3Primary,
          secondaryContainer: _m3Container,
          onSecondaryContainer: _m3Primary,
          tertiary: _m3Accent,
          onTertiary: _m3Primary,
          tertiaryContainer: _m3Container,
          onTertiaryContainer: _m3Primary,
          surface: _m3Surface,
          onSurface: _m3Primary,
          surfaceContainerLowest: Colors.white,
          surfaceContainerLow: _m3Surface,
          surfaceContainer: _m3Container.withValues(alpha: 0.38),
          surfaceContainerHigh: _m3Container.withValues(alpha: 0.54),
          surfaceContainerHighest: _m3Container.withValues(alpha: 0.72),
          outline: _m3Accent,
          outlineVariant: _m3Container,
          error: const Color(0xFFB3261E),
          onError: Colors.white,
        );
  final baseTextTheme = isDark
      ? Typography.material2021().white
      : Typography.material2021().black;

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    scaffoldBackgroundColor: scheme.surface,
    appBarTheme: AppBarTheme(
      centerTitle: false,
      backgroundColor: Colors.transparent,
      foregroundColor: scheme.onSurface,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      titleTextStyle: TextStyle(
        fontSize: 20,
        fontWeight: FontWeight.w700,
        color: scheme.onSurface,
        letterSpacing: -0.2,
      ),
    ),
    cardTheme: CardThemeData(
      elevation: 0,
      color: scheme.surfaceContainerLow,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(28),
        side: BorderSide(
          color: scheme.outlineVariant.withValues(alpha: isDark ? 0.45 : 0.8),
        ),
      ),
    ),
    chipTheme: ChipThemeData(
      side: BorderSide.none,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        minimumSize: const Size(0, 52),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        minimumSize: const Size(0, 52),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
      ),
    ),
    textTheme: baseTextTheme.apply(
      bodyColor: scheme.onSurface,
      displayColor: scheme.onSurface,
    ),
  );
}

class ServerPickerPage extends StatefulWidget {
  const ServerPickerPage({required this.onOpenServer, super.key});

  final ValueChanged<DiscoveredServer> onOpenServer;

  @override
  State<ServerPickerPage> createState() => _ServerPickerPageState();
}

class _ServerPickerPageState extends State<ServerPickerPage> {
  final http.Client _client = http.Client();

  bool _scanning = false;
  DiscoveryResult? _result;

  @override
  void initState() {
    super.initState();
    unawaited(_scan());
  }

  @override
  void dispose() {
    _client.close();
    super.dispose();
  }

  Future<void> _scan() async {
    if (_scanning) {
      return;
    }

    setState(() {
      _scanning = true;
    });

    final preferredEndpoint = await loadLastUsedServer();
    final result = await discoverServers(
      _client,
      preferredEndpoint: preferredEndpoint,
    );
    if (!mounted) {
      return;
    }

    setState(() {
      _result = result;
      _scanning = false;
    });
  }

  Future<void> _openManualEntrySheet() async {
    final server = await showModalBottomSheet<DiscoveredServer>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) => ManualServerSheet(client: _client),
    );
    if (server == null || !mounted) {
      return;
    }
    widget.onOpenServer(server);
  }

  @override
  Widget build(BuildContext context) {
    final servers = _result?.servers ?? const <DiscoveredServer>[];

    return Scaffold(
      appBar: AppBar(
        title: const Text('gscale-zebra'),
        actions: [
          IconButton(
            onPressed: _openManualEntrySheet,
            icon: const Icon(Icons.add_link_rounded),
            tooltip: 'Add',
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _scan,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(18, 12, 18, 24),
          children: [
            if (_scanning && servers.isEmpty) const _ScanningState(),
            if (!_scanning && servers.isEmpty)
              _EmptyServerState(onManualAdd: _openManualEntrySheet),
            if (servers.isNotEmpty)
              _ServerList(servers: servers, onOpenServer: widget.onOpenServer),
          ],
        ),
      ),
    );
  }
}

class OperatorDashboardPage extends StatefulWidget {
  const OperatorDashboardPage({
    required this.server,
    required this.onChangeServer,
    super.key,
  });

  final DiscoveredServer server;
  final VoidCallback onChangeServer;

  @override
  State<OperatorDashboardPage> createState() => _OperatorDashboardPageState();
}

class _OperatorDashboardPageState extends State<OperatorDashboardPage> {
  final http.Client _client = http.Client();
  StreamSubscription<String>? _streamSubscription;
  int _streamGeneration = 0;
  late DiscoveredServer _server;
  int _selectedSection = 0;

  bool _manualLoading = false;
  bool _requestInFlight = false;
  bool _connected = false;
  String _statusText = 'idle';
  String _errorText = '';
  MonitorSnapshot _snapshot = MonitorSnapshot.empty();

  @override
  void initState() {
    super.initState();
    _server = widget.server;
    _startLiveStream();
  }

  @override
  void didUpdateWidget(covariant OperatorDashboardPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.server.endpoint.baseUrl != widget.server.endpoint.baseUrl) {
      _server = widget.server;
    }
  }

  @override
  void dispose() {
    _stopLiveStream();
    _client.close();
    super.dispose();
  }

  Future<void> _editDisplayName() async {
    final controller = TextEditingController(
      text: _server.handshake.displayName,
    );
    final nickname = await showDialog<String>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('Server name'),
          content: TextField(
            controller: controller,
            autofocus: true,
            textInputAction: TextInputAction.done,
            decoration: const InputDecoration(
              hintText: 'Enter server name',
              border: OutlineInputBorder(),
            ),
            onSubmitted: (_) =>
                Navigator.of(context).pop(controller.text.trim()),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () =>
                  Navigator.of(context).pop(controller.text.trim()),
              child: const Text('Save'),
            ),
          ],
        );
      },
    );
    controller.dispose();

    final trimmed = nickname?.trim() ?? '';
    if (trimmed.isEmpty || trimmed == _server.handshake.displayName) {
      return;
    }

    try {
      final response = await _client
          .put(
            Uri.parse('${_server.endpoint.baseUrl}/v1/mobile/profile'),
            headers: const {'Content-Type': 'application/json'},
            body: jsonEncode({'nickname': trimmed}),
          )
          .timeout(const Duration(seconds: 4));
      if (response.statusCode < 200 || response.statusCode > 299) {
        throw Exception('profile ${response.statusCode}');
      }
      final payload = jsonDecode(response.body) as Map<String, dynamic>;
      if (!mounted) {
        return;
      }
      setState(() {
        _server = _server.copyWith(
          handshake: _server.handshake.copyWith(
            displayName: _text(payload['display_name'], fallback: trimmed),
          ),
        );
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  void _startLiveStream() {
    _streamGeneration++;
    final generation = _streamGeneration;
    unawaited(_runLiveStream(generation));
  }

  void _stopLiveStream() {
    _streamGeneration++;
    unawaited(_streamSubscription?.cancel());
    _streamSubscription = null;
  }

  Future<void> _runLiveStream(int generation) async {
    while (mounted && generation == _streamGeneration) {
      try {
        if (mounted) {
          setState(() {
            _statusText = _connected ? 'reconnecting' : 'connecting';
          });
        }
        await _connectLiveStreamOnce(generation);
      } catch (error) {
        if (!mounted || generation != _streamGeneration) {
          return;
        }
        setState(() {
          _connected = false;
          _statusText = 'offline';
          _errorText = error.toString();
        });
      }

      if (!mounted || generation != _streamGeneration) {
        return;
      }
      await Future.delayed(const Duration(seconds: 1));
    }
  }

  Future<void> _connectLiveStreamOnce(int generation) async {
    final request = http.Request(
      'GET',
      Uri.parse('${widget.server.endpoint.baseUrl}/v1/mobile/monitor/stream'),
    );
    request.headers['Accept'] = 'text/event-stream';

    final response = await _client
        .send(request)
        .timeout(const Duration(seconds: 4));
    if (response.statusCode < 200 || response.statusCode > 299) {
      throw Exception('stream ${response.statusCode}');
    }

    final completer = Completer<void>();
    final dataLines = <String>[];

    await _streamSubscription?.cancel();
    _streamSubscription = response.stream
        .transform(utf8.decoder)
        .transform(const LineSplitter())
        .listen(
          (line) {
            if (!mounted || generation != _streamGeneration) {
              return;
            }
            if (line.isEmpty) {
              if (dataLines.isEmpty) {
                return;
              }
              final payloadText = dataLines.join('\n');
              dataLines.clear();
              final payload = jsonDecode(payloadText) as Map<String, dynamic>;
              if (payload.containsKey('error') && payload['ok'] != true) {
                setState(() {
                  _connected = false;
                  _statusText = 'offline';
                  _errorText = payload['error'].toString();
                });
                return;
              }
              setState(() {
                _snapshot = MonitorSnapshot.fromJson(payload);
                _connected = true;
                _statusText = 'live';
                _errorText = '';
              });
              return;
            }
            if (line.startsWith(':')) {
              return;
            }
            if (line.startsWith('data:')) {
              dataLines.add(line.substring(5).trimLeft());
            }
          },
          onError: (error, _) {
            if (!completer.isCompleted) {
              completer.completeError(error);
            }
          },
          onDone: () {
            if (!completer.isCompleted) {
              completer.complete();
            }
          },
          cancelOnError: true,
        );

    await completer.future;
  }

  Future<void> _refresh({bool manual = false}) async {
    if (_requestInFlight) {
      return;
    }

    _requestInFlight = true;
    if (manual && mounted) {
      setState(() {
        _manualLoading = true;
        _errorText = '';
        _statusText = 'refreshing';
      });
    }

    try {
      final health = await _client
          .get(Uri.parse('${widget.server.endpoint.baseUrl}/healthz'))
          .timeout(const Duration(seconds: 4));
      if (health.statusCode < 200 || health.statusCode > 299) {
        throw Exception('healthz ${health.statusCode}');
      }

      final monitor = await _client
          .get(
            Uri.parse(
              '${widget.server.endpoint.baseUrl}/v1/mobile/monitor/state',
            ),
          )
          .timeout(const Duration(seconds: 4));
      if (monitor.statusCode < 200 || monitor.statusCode > 299) {
        throw Exception('monitor ${monitor.statusCode}');
      }

      final payload = jsonDecode(monitor.body) as Map<String, dynamic>;
      if (mounted) {
        setState(() {
          _snapshot = MonitorSnapshot.fromJson(payload);
          _connected = true;
          _statusText = 'connected';
          _errorText = '';
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _connected = false;
          _statusText = 'offline';
          _errorText = error.toString();
        });
      }
    } finally {
      _requestInFlight = false;
      if (manual && mounted) {
        setState(() {
          _manualLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final server = _server;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          onPressed: widget.onChangeServer,
          icon: const Icon(Icons.arrow_back_rounded),
          tooltip: 'Change server',
        ),
        title: Text(server.handshake.serverName),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedSection,
        onDestinationSelected: (index) {
          setState(() {
            _selectedSection = index;
          });
        },
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.monitor_heart_outlined),
            label: 'Status',
          ),
          NavigationDestination(
            icon: Icon(Icons.tune_rounded),
            label: 'Control',
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(18, 8, 18, 24),
        children: [
          Row(
            children: [
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 7,
                ),
                decoration: BoxDecoration(
                  color: _connected
                      ? scheme.secondaryContainer
                      : scheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  _connected ? 'Connected' : 'Offline',
                  style: theme.textTheme.labelLarge?.copyWith(
                    color: _connected
                        ? scheme.onSecondaryContainer
                        : scheme.onSurfaceVariant,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              const Spacer(),
              Text(
                server.latencyMs > 0 ? '${server.latencyMs} ms' : _statusText,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: scheme.onSurfaceVariant,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Expanded(
                child: Text(
                  server.handshake.displayName,
                  style: theme.textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w800,
                    letterSpacing: -0.3,
                  ),
                ),
              ),
              IconButton(
                onPressed: _editDisplayName,
                icon: const Icon(Icons.edit_outlined),
                tooltip: 'Rename server',
              ),
            ],
          ),
          Text(
            server.endpoint.baseUrl,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: scheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            _connected ? _snapshot.serverLabel : 'API: offline',
            style: theme.textTheme.bodySmall?.copyWith(
              color: scheme.onSurfaceVariant,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 22),
          Text(
            'Line overview',
            style: theme.textTheme.titleLarge?.copyWith(
              fontWeight: FontWeight.w800,
              letterSpacing: -0.2,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            _connected
                ? 'Tanlangan serverdan live holat olindi.'
                : 'Pastdagi refresh bilan server snapshot qayta olinadi.',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: scheme.onSurfaceVariant,
            ),
          ),
          if (_errorText.isNotEmpty) ...[
            const SizedBox(height: 12),
            Text(
              _errorText,
              style: theme.textTheme.bodySmall?.copyWith(color: scheme.error),
            ),
          ],
          const SizedBox(height: 16),
          ...(_selectedSection == 0
              ? _buildStatusSection()
              : _buildControlSection()),
        ],
      ),
    );
  }

  bool get _showMonitorRow {
    if (!_connected) {
      return false;
    }
    final label = _snapshot.monitorLabel.trim();
    if (label.isEmpty) {
      return false;
    }
    if (label == 'No active batch') {
      return false;
    }
    return true;
  }

  List<Widget> _buildStatusSection() {
    return [
      _StatusSummary(snapshot: _snapshot),
      if (_showMonitorRow) ...[
        const SizedBox(height: 20),
        _InfoList(
          rows: [
            _InfoRowData(
              icon: Icons.monitor_heart_outlined,
              title: 'Monitor',
              subtitle: _snapshot.monitorLabel,
            ),
          ],
        ),
      ],
    ];
  }

  List<Widget> _buildControlSection() {
    return [
      Text(
        'Control',
        style: Theme.of(context).textTheme.titleLarge?.copyWith(
          fontWeight: FontWeight.w800,
          letterSpacing: -0.2,
        ),
      ),
      const SizedBox(height: 4),
      Text(
        'Printer va server boshqaruvi.',
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
      const SizedBox(height: 16),
      _InfoList(
        rows: [
          _InfoRowData(
            icon: Icons.tune,
            title: 'Printer',
            subtitle: _connected
                ? _snapshot.printerLabel
                : 'Printer trace va action holati',
          ),
        ],
      ),
      const SizedBox(height: 18),
      Row(
        children: [
          Expanded(
            child: FilledButton.icon(
              onPressed: _manualLoading ? null : () => _refresh(manual: true),
              icon: const Icon(Icons.refresh_rounded),
              label: Text(_manualLoading ? 'Refreshing...' : 'Refresh'),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: OutlinedButton.icon(
              onPressed: widget.onChangeServer,
              icon: const Icon(Icons.dns_rounded),
              label: const Text('Servers'),
            ),
          ),
        ],
      ),
      const SizedBox(height: 10),
      OutlinedButton.icon(
        onPressed: _editDisplayName,
        icon: const Icon(Icons.edit_outlined),
        label: const Text('Rename server'),
      ),
    ];
  }
}

class _ScanningState extends StatelessWidget {
  const _ScanningState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 8),
      child: Row(
        children: [
          const SizedBox(
            width: 24,
            height: 24,
            child: CircularProgressIndicator(strokeWidth: 2.6),
          ),
          const SizedBox(width: 14),
          Text(
            'Scanning...',
            style: theme.textTheme.bodyLarge?.copyWith(
              color: scheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyServerState extends StatelessWidget {
  const _EmptyServerState({required this.onManualAdd});

  final VoidCallback onManualAdd;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'No servers',
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w800,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            'Pull down to refresh or add address.',
            style: theme.textTheme.bodyMedium?.copyWith(
              color: scheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 12),
          TextButton(onPressed: onManualAdd, child: const Text('Add address')),
        ],
      ),
    );
  }
}

class _ServerCard extends StatelessWidget {
  const _ServerCard({required this.server, required this.onOpen});

  final DiscoveredServer server;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return ListTile(
      onTap: onOpen,
      dense: false,
      contentPadding: const EdgeInsets.symmetric(horizontal: 6, vertical: 6),
      leading: Icon(
        _wifiIconForLatency(server.latencyMs),
        color: scheme.primary,
        size: 28,
      ),
      title: Text(
        server.handshake.serverName,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: theme.textTheme.titleMedium?.copyWith(
          fontWeight: FontWeight.w700,
        ),
      ),
      subtitle: Text(
        server.endpoint.label,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: scheme.onSurfaceVariant,
        ),
      ),
      trailing: Text(
        'Connect',
        style: theme.textTheme.labelLarge?.copyWith(
          color: scheme.primary,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

class _ServerList extends StatelessWidget {
  const _ServerList({required this.servers, required this.onOpenServer});

  final List<DiscoveredServer> servers;
  final ValueChanged<DiscoveredServer> onOpenServer;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return Column(
      children: [
        for (var i = 0; i < servers.length; i++) ...[
          _ServerCard(
            server: servers[i],
            onOpen: () => onOpenServer(servers[i]),
          ),
          if (i != servers.length - 1)
            Divider(
              height: 1,
              indent: 52,
              endIndent: 6,
              color: scheme.outlineVariant,
            ),
        ],
      ],
    );
  }
}

IconData _wifiIconForLatency(int latencyMs) {
  if (latencyMs <= 8) {
    return Icons.signal_wifi_4_bar_rounded;
  }
  if (latencyMs <= 25) {
    return Icons.network_wifi_3_bar_rounded;
  }
  if (latencyMs <= 60) {
    return Icons.network_wifi_2_bar_rounded;
  }
  return Icons.network_wifi_1_bar_rounded;
}

class ManualServerSheet extends StatefulWidget {
  const ManualServerSheet({required this.client, super.key});

  final http.Client client;

  @override
  State<ManualServerSheet> createState() => _ManualServerSheetState();
}

class _ManualServerSheetState extends State<ManualServerSheet> {
  late final TextEditingController _controller;
  bool _checking = false;
  String _errorText = '';

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(text: _defaultApiBaseUrl);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_checking) {
      return;
    }

    setState(() {
      _checking = true;
      _errorText = '';
    });

    final endpoint = parseServerEndpoint(_controller.text);
    if (endpoint == null) {
      setState(() {
        _checking = false;
        _errorText = 'Address format is invalid';
      });
      return;
    }

    final server = await probeServer(
      widget.client,
      endpoint,
      timeout: _manualProbeTimeout,
    );
    if (!mounted) {
      return;
    }

    if (server == null) {
      setState(() {
        _checking = false;
        _errorText = 'Handshake failed for this server';
      });
      return;
    }

    Navigator.of(context).pop(server);
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;

    return Padding(
      padding: EdgeInsets.fromLTRB(18, 0, 18, bottomInset + 18),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Add server',
            style: Theme.of(
              context,
            ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
          ),
          const SizedBox(height: 8),
          Text(
            'Example: 192.168.1.12:8081',
            style: Theme.of(
              context,
            ).textTheme.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _controller,
            keyboardType: TextInputType.url,
            decoration: const InputDecoration(
              labelText: 'Server address',
              hintText: 'http://192.168.1.12:8081',
              border: OutlineInputBorder(),
            ),
            onSubmitted: (_) => _submit(),
          ),
          if (_errorText.isNotEmpty) ...[
            const SizedBox(height: 10),
            Text(_errorText, style: TextStyle(color: scheme.error)),
          ],
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: _checking ? null : _submit,
            icon: const Icon(Icons.link_rounded),
            label: Text(_checking ? 'Checking...' : 'Connect to server'),
          ),
        ],
      ),
    );
  }
}

class _StatusSummary extends StatelessWidget {
  const _StatusSummary({required this.snapshot});

  final MonitorSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return Column(
      children: [
        _StatusRow(
          icon: Icons.scale_outlined,
          title: 'Scale',
          value: snapshot.scaleValue,
          caption: snapshot.scaleCaption,
        ),
        Divider(height: 20, color: scheme.outlineVariant),
        _StatusRow(
          icon: Icons.print_outlined,
          title: 'Zebra',
          value: snapshot.zebraValue,
          caption: snapshot.zebraCaption,
        ),
        Divider(height: 20, color: scheme.outlineVariant),
        _StatusRow(
          icon: Icons.inventory_2_outlined,
          title: 'Batch',
          value: snapshot.batchValue,
          caption: snapshot.batchCaption,
        ),
        Divider(height: 20, color: scheme.outlineVariant),
        _StatusRow(
          icon: Icons.sync_outlined,
          title: 'Bridge',
          value: snapshot.bridgeValue,
          caption: snapshot.bridgeCaption,
        ),
      ],
    );
  }
}

class _StatusRow extends StatelessWidget {
  const _StatusRow({
    required this.icon,
    required this.title,
    required this.value,
    required this.caption,
  });

  final IconData icon;
  final String title;
  final String value;
  final String caption;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Row(
      children: [
        Container(
          width: 42,
          height: 42,
          decoration: BoxDecoration(
            color: scheme.surfaceContainerHighest.withValues(alpha: 0.5),
            borderRadius: BorderRadius.circular(14),
          ),
          child: Icon(icon, color: scheme.primary, size: 20),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: theme.textTheme.labelLarge?.copyWith(
                  color: scheme.onSurfaceVariant,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                value,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w800,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 12),
        Flexible(
          child: Text(
            caption,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.right,
            style: theme.textTheme.bodySmall?.copyWith(
              color: scheme.onSurfaceVariant,
              height: 1.2,
            ),
          ),
        ),
      ],
    );
  }
}

class _InfoRowData {
  const _InfoRowData({
    required this.icon,
    required this.title,
    required this.subtitle,
  });

  final IconData icon;
  final String title;
  final String subtitle;
}

class _InfoList extends StatelessWidget {
  const _InfoList({required this.rows});

  final List<_InfoRowData> rows;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Column(
      children: [
        for (var i = 0; i < rows.length; i++) ...[
          _TodoRow(
            icon: rows[i].icon,
            title: rows[i].title,
            subtitle: rows[i].subtitle,
          ),
          if (i != rows.length - 1)
            Divider(height: 24, color: scheme.outlineVariant),
        ],
      ],
    );
  }
}

class _TodoRow extends StatelessWidget {
  const _TodoRow({
    required this.icon,
    required this.title,
    required this.subtitle,
  });

  final IconData icon;
  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 38,
          height: 38,
          decoration: BoxDecoration(
            color: scheme.secondaryContainer,
            borderRadius: BorderRadius.circular(14),
          ),
          child: Icon(icon, color: scheme.onSecondaryContainer),
        ),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: theme.textTheme.bodyLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                subtitle,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: scheme.onSurfaceVariant,
                  height: 1.3,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class MonitorSnapshot {
  const MonitorSnapshot({
    required this.scaleValue,
    required this.scaleCaption,
    required this.zebraValue,
    required this.zebraCaption,
    required this.batchValue,
    required this.batchCaption,
    required this.bridgeValue,
    required this.bridgeCaption,
    required this.serverLabel,
    required this.monitorLabel,
    required this.printerLabel,
  });

  factory MonitorSnapshot.empty() {
    return const MonitorSnapshot(
      scaleValue: '--',
      scaleCaption: 'Live qty',
      zebraValue: 'Idle',
      zebraCaption: 'Printer state',
      batchValue: 'Stopped',
      batchCaption: 'Workflow',
      bridgeValue: 'Ready',
      bridgeCaption: 'Shared state',
      serverLabel: 'API: idle',
      monitorLabel: 'Scale, Zebra, batch va print request holati',
      printerLabel: 'Printer trace va action holati',
    );
  }

  factory MonitorSnapshot.fromJson(Map<String, dynamic> json) {
    final state = (json['state'] as Map?)?.cast<String, dynamic>() ?? const {};
    final scale = (state['scale'] as Map?)?.cast<String, dynamic>() ?? const {};
    final zebra = (state['zebra'] as Map?)?.cast<String, dynamic>() ?? const {};
    final batch = (state['batch'] as Map?)?.cast<String, dynamic>() ?? const {};
    final printRequest =
        (state['print_request'] as Map?)?.cast<String, dynamic>() ?? const {};
    final printer =
        (json['printer'] as Map?)?.cast<String, dynamic>() ?? const {};

    final scaleWeight = scale['weight'];
    final scaleUnit = _text(scale['unit'], fallback: 'kg');
    final scaleStable = scale['stable'] == true ? 'stable' : 'live';

    final zebraVerify = _text(zebra['verify'], fallback: 'idle');
    final zebraAction = _text(zebra['action'], fallback: 'printer state');

    final batchActive = batch['active'] == true;
    final batchItem = _text(
      batch['item_name'],
      fallback: _text(batch['item_code']),
    );

    final printStatus = _text(printRequest['status'], fallback: 'idle');
    final printerMode = _text(
      printer['print_mode'],
      fallback: 'trace unavailable',
    );

    return MonitorSnapshot(
      scaleValue: scaleWeight == null ? '--' : '$scaleWeight $scaleUnit',
      scaleCaption: scaleStable,
      zebraValue: zebraVerify.toUpperCase(),
      zebraCaption: zebraAction,
      batchValue: batchActive ? 'Active' : 'Stopped',
      batchCaption: batchItem.isEmpty ? 'Workflow' : batchItem,
      bridgeValue: printStatus == 'idle' ? 'Ready' : printStatus,
      bridgeCaption: _text(printRequest['epc'], fallback: 'Shared state'),
      serverLabel: _text(json['ok'], fallback: 'unknown') == 'true'
          ? 'API: online'
          : 'API: offline',
      monitorLabel: batchItem.isEmpty ? 'No active batch' : 'Batch: $batchItem',
      printerLabel: 'Print mode: $printerMode',
    );
  }

  final String scaleValue;
  final String scaleCaption;
  final String zebraValue;
  final String zebraCaption;
  final String batchValue;
  final String batchCaption;
  final String bridgeValue;
  final String bridgeCaption;
  final String serverLabel;
  final String monitorLabel;
  final String printerLabel;
}

class DiscoveryResult {
  const DiscoveryResult({required this.servers, required this.candidateCount});

  final List<DiscoveredServer> servers;
  final int candidateCount;
}

class DiscoveredServer {
  const DiscoveredServer({
    required this.endpoint,
    required this.handshake,
    required this.latencyMs,
  });

  final ServerEndpoint endpoint;
  final ServerHandshake handshake;
  final int latencyMs;

  String get discoveryKey {
    final ref = handshake.serverRef.trim().toLowerCase();
    final name = handshake.serverName.trim().toLowerCase();
    if (ref.isNotEmpty && ref != 'unknown' && ref != 'legacy-healthz') {
      return '$ref|$name';
    }
    return endpoint.label.toLowerCase();
  }

  DiscoveredServer copyWith({
    ServerEndpoint? endpoint,
    ServerHandshake? handshake,
    int? latencyMs,
  }) {
    return DiscoveredServer(
      endpoint: endpoint ?? this.endpoint,
      handshake: handshake ?? this.handshake,
      latencyMs: latencyMs ?? this.latencyMs,
    );
  }
}

class ServerEndpoint {
  const ServerEndpoint({
    required this.host,
    required this.port,
    required this.baseUrl,
  });

  final String host;
  final int port;
  final String baseUrl;

  String get label => '$host:$port';
}

class ServerHandshake {
  const ServerHandshake({
    required this.serverName,
    required this.displayName,
    required this.role,
    required this.serverRef,
  });

  factory ServerHandshake.fromJson(Map<String, dynamic> json) {
    return ServerHandshake(
      serverName: _text(json['server_name'], fallback: 'gscale-zebra'),
      displayName: _text(json['display_name'], fallback: 'Operator'),
      role: _text(json['role'], fallback: 'operator'),
      serverRef: _text(json['server_ref'], fallback: 'unknown'),
    );
  }

  final String serverName;
  final String displayName;
  final String role;
  final String serverRef;

  ServerHandshake copyWith({
    String? serverName,
    String? displayName,
    String? role,
    String? serverRef,
  }) {
    return ServerHandshake(
      serverName: serverName ?? this.serverName,
      displayName: displayName ?? this.displayName,
      role: role ?? this.role,
      serverRef: serverRef ?? this.serverRef,
    );
  }
}

Future<DiscoveryResult> discoverServers(
  http.Client client, {
  ServerEndpoint? preferredEndpoint,
}) async {
  final announcementsFuture = network_candidates.discoverAnnouncements(
    port: _discoveryPort,
    timeout: _udpDiscoveryTimeout,
  );
  final candidates = await network_candidates.collectCandidateHosts();
  final resultsByKey = <String, DiscoveredServer>{};

  final probeTargets = <ServerEndpoint>[];
  if (preferredEndpoint != null) {
    probeTargets.add(preferredEndpoint);
  }
  for (final host in candidates) {
    final endpoint = ServerEndpoint(
      host: host,
      port: _defaultApiPort,
      baseUrl: 'http://$host:$_defaultApiPort',
    );
    if (probeTargets.any((item) => item.baseUrl == endpoint.baseUrl)) {
      continue;
    }
    probeTargets.add(endpoint);
  }

  final directScanned = await Future.wait(
    probeTargets.map((endpoint) {
      return probeServer(client, endpoint, timeout: _fastProbeTimeout);
    }),
  );
  for (final server in directScanned.whereType<DiscoveredServer>()) {
    final existing = resultsByKey[server.discoveryKey];
    if (existing == null || server.latencyMs < existing.latencyMs) {
      resultsByKey[server.discoveryKey] = server;
    }
  }

  final announcements = await announcementsFuture;
  for (final announcement in announcements) {
    final server = DiscoveredServer(
      endpoint: ServerEndpoint(
        host: announcement.host,
        port: announcement.httpPort,
        baseUrl: 'http://${announcement.host}:${announcement.httpPort}',
      ),
      handshake: ServerHandshake(
        serverName: announcement.serverName,
        displayName: announcement.displayName,
        role: announcement.role,
        serverRef: announcement.serverRef,
      ),
      latencyMs: announcement.latencyMs,
    );
    final existing = resultsByKey[server.discoveryKey];
    if (existing == null || server.latencyMs < existing.latencyMs) {
      resultsByKey[server.discoveryKey] = server;
    }
  }

  final results = resultsByKey.values.toList();

  results.sort((left, right) {
    if (preferredEndpoint != null) {
      final leftPreferred = left.endpoint.baseUrl == preferredEndpoint.baseUrl;
      final rightPreferred =
          right.endpoint.baseUrl == preferredEndpoint.baseUrl;
      if (leftPreferred != rightPreferred) {
        return leftPreferred ? -1 : 1;
      }
    }
    final latencyCmp = left.latencyMs.compareTo(right.latencyMs);
    if (latencyCmp != 0) {
      return latencyCmp;
    }
    return left.endpoint.baseUrl.compareTo(right.endpoint.baseUrl);
  });

  return DiscoveryResult(servers: results, candidateCount: candidates.length);
}

Future<DiscoveredServer?> probeServer(
  http.Client client,
  ServerEndpoint endpoint, {
  Duration timeout = _fastProbeTimeout,
}) async {
  final stopwatch = Stopwatch()..start();

  try {
    final handshakeResponse = await client
        .get(Uri.parse('${endpoint.baseUrl}/v1/mobile/handshake'))
        .timeout(timeout);
    if (handshakeResponse.statusCode >= 200 &&
        handshakeResponse.statusCode < 300) {
      final json = jsonDecode(handshakeResponse.body) as Map<String, dynamic>;
      if (_text(json['service']) != 'mobileapi') {
        return null;
      }
      stopwatch.stop();
      return DiscoveredServer(
        endpoint: endpoint,
        handshake: ServerHandshake.fromJson(json),
        latencyMs: stopwatch.elapsedMilliseconds,
      );
    }

    final healthResponse = await client
        .get(Uri.parse('${endpoint.baseUrl}/healthz'))
        .timeout(timeout);
    if (healthResponse.statusCode < 200 || healthResponse.statusCode > 299) {
      return null;
    }

    final health = jsonDecode(healthResponse.body) as Map<String, dynamic>;
    if (_text(health['service']) != 'mobileapi') {
      return null;
    }

    stopwatch.stop();
    return DiscoveredServer(
      endpoint: endpoint,
      handshake: ServerHandshake(
        serverName: endpoint.host,
        displayName: 'Operator',
        role: 'operator',
        serverRef: 'legacy-healthz',
      ),
      latencyMs: stopwatch.elapsedMilliseconds,
    );
  } catch (_) {
    return null;
  }
}

ServerEndpoint? parseServerEndpoint(String raw) {
  var value = raw.trim();
  if (value.isEmpty) {
    return null;
  }
  if (!value.contains('://')) {
    value = 'http://$value';
  }

  final uri = Uri.tryParse(value);
  if (uri == null || (uri.host.isEmpty && uri.path.isEmpty)) {
    return null;
  }

  final host = uri.host.isNotEmpty ? uri.host : uri.path;
  if (host.trim().isEmpty) {
    return null;
  }

  final port = uri.hasPort ? uri.port : _defaultApiPort;
  final scheme = uri.scheme.isEmpty ? 'http' : uri.scheme;
  return ServerEndpoint(
    host: host,
    port: port,
    baseUrl: '$scheme://$host:$port',
  );
}

String _text(Object? value, {String fallback = ''}) {
  final text = value?.toString().trim() ?? '';
  if (text.isEmpty) {
    return fallback;
  }
  return text;
}

Future<void> saveLastUsedServer(ServerEndpoint endpoint) async {
  final prefs = await SharedPreferences.getInstance();
  await prefs.setString(_lastServerKey, endpoint.baseUrl);
}

Future<ServerEndpoint?> loadLastUsedServer() async {
  final prefs = await SharedPreferences.getInstance();
  final value = prefs.getString(_lastServerKey);
  if (value == null || value.trim().isEmpty) {
    return null;
  }
  return parseServerEndpoint(value);
}
