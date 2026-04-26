import SwiftUI
import PhotosUI

struct ScannerView: View {
    @StateObject private var cameraService = CameraService()
    @StateObject private var vm = ScannerViewModel()

    @State private var showResult = false
    @State private var selectedPhoto: PhotosPickerItem? = nil

    var body: some View {
        NavigationStack {
            ZStack {
                CameraPreviewView(session: cameraService.session)
                    .ignoresSafeArea()

                VStack {
                    Spacer()
                    statusBar
                    controlBar
                }
            }
            .onAppear {
                vm.bind(cameraService: cameraService)
                cameraService.checkPermissionAndStart()
            }
            .onDisappear { cameraService.stopSession() }
            .alert("カメラのアクセスが拒否されています", isPresented: $cameraService.permissionDenied) {
                Button("設定を開く") {
                    if let url = URL(string: UIApplication.openSettingsURLString) {
                        UIApplication.shared.open(url)
                    }
                }
                Button("キャンセル", role: .cancel) {}
            } message: {
                Text("設定アプリからカメラへのアクセスを許可してください")
            }
            .sheet(isPresented: $showResult) {
                if let result = vm.parseResult {
                    ResultView(parseResult: result)
                }
            }
            .onChange(of: selectedPhoto) { newItem in
                guard let newItem else { return }
                Task {
                    if let data = try? await newItem.loadTransferable(type: Data.self),
                       let image = UIImage(data: data) {
                        vm.processImage(image)
                    }
                }
            }
            .navigationTitle("yakuqr")
            .navigationBarTitleDisplayMode(.inline)
        }
    }

    private var statusBar: some View {
        HStack {
            Text("読み取り済み: \(vm.scannedQRs.count) 件")
                .font(.subheadline.bold())
                .foregroundStyle(.white)
                .padding(.horizontal, 12)
                .padding(.vertical, 6)
                .background(.black.opacity(0.6))
                .clipShape(Capsule())
        }
        .padding(.bottom, 8)
    }

    private var controlBar: some View {
        HStack(spacing: 20) {
            PhotosPicker(selection: $selectedPhoto, matching: .images) {
                Image(systemName: "photo.on.rectangle")
                    .font(.title2)
                    .foregroundStyle(.white)
                    .frame(width: 56, height: 56)
                    .background(.black.opacity(0.6))
                    .clipShape(Circle())
            }

            Button {
                vm.parse()
                showResult = true
            } label: {
                Text("解析する")
                    .font(.headline)
                    .foregroundStyle(.white)
                    .padding(.horizontal, 32)
                    .padding(.vertical, 14)
                    .background(vm.scannedQRs.isEmpty ? Color.gray : Color.blue)
                    .clipShape(Capsule())
            }

            Button {
                vm.reset()
            } label: {
                Image(systemName: "arrow.counterclockwise")
                    .font(.title2)
                    .foregroundStyle(.white)
                    .frame(width: 56, height: 56)
                    .background(.black.opacity(0.6))
                    .clipShape(Circle())
            }
        }
        .padding(.bottom, 40)
    }
}
