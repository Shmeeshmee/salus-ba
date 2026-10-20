import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

/// Shows whether the API answers its liveness check.
class HealthScreen extends StatefulWidget {
  const HealthScreen({super.key, required this.api});

  final HealthApi api;

  @override
  State<HealthScreen> createState() => _HealthScreenState();
}

class _HealthScreenState extends State<HealthScreen> {
  late final Future<String> _status = _check();

  Future<String> _check() async {
    try {
      final health = await widget.api.getHealth();
      return health?.status == StatusStatusEnum.ok ? 'ok' : 'unavailable';
    } catch (_) {
      return 'unavailable';
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Salus')),
      body: Center(
        child: FutureBuilder<String>(
          future: _status,
          builder: (context, snapshot) => Text('API status: ${snapshot.data ?? 'checking'}'),
        ),
      ),
    );
  }
}
