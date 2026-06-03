extension ProtosString on String {
  String? get nonEmpty {
    final trimmed = trim();
    return trimmed.isEmpty ? null : trimmed;
  }

  Map<String, String> get credentialPairs {
    final pairs = <String, String>{};
    for (final rawPair in split(RegExp(r'[\n,]'))) {
      final parts = rawPair.split('=');
      if (parts.length < 2) {
        continue;
      }
      final key = parts.first.trim();
      final value = parts.sublist(1).join('=').trim();
      if (key.isNotEmpty) {
        pairs[key] = value;
      }
    }
    return pairs;
  }

  List<String> get routeCidrs {
    return split(RegExp(r'[\n,]'))
        .map((value) => value.trim())
        .where((value) => value.isNotEmpty)
        .toList(growable: false);
  }
}

String fallbackText(String? value) => value?.nonEmpty ?? 'n/a';

String? shortHash(String? value, {int length = 12}) {
  final trimmed = value?.nonEmpty;
  if (trimmed == null || trimmed.length <= length) {
    return trimmed;
  }
  return trimmed.substring(0, length);
}
