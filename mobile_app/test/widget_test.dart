import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/main.dart';

void main() {
  testWidgets('renders server picker shell', (tester) async {
    await tester.pumpWidget(const GScaleMobileApp());

    expect(find.text('gscale-zebra'), findsOneWidget);
    expect(find.byIcon(Icons.add_link_rounded), findsOneWidget);
  });
}
