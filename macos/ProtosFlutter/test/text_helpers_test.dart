import 'package:flutter_test/flutter_test.dart';
import 'package:protos_flutter/src/text_helpers.dart';

void main() {
  test('parses credential pairs from newlines and commas', () {
    expect('TOKEN=abc\nregion=eu, empty , nested=a=b'.credentialPairs, {
      'TOKEN': 'abc',
      'region': 'eu',
      'nested': 'a=b',
    });
  });

  test('parses route cidrs from newlines and commas', () {
    expect('0.0.0.0/0, 10.0.0.0/8\n\n192.168.0.0/16'.routeCidrs, [
      '0.0.0.0/0',
      '10.0.0.0/8',
      '192.168.0.0/16',
    ]);
  });
}
