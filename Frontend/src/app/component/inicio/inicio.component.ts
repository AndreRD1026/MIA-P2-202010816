import { Component } from '@angular/core';
import { ProyectoService } from 'src/app/services/proyecto.service';
//import Swal from 'sweetalert2';

@Component({
  selector: 'app-inicio',
  templateUrl: './inicio.component.html',
  styleUrls: ['./inicio.component.css']
})
export class InicioComponent {

  entrada = "";
  salida = "";

  constructor(private service: ProyectoService) { }

  ngOnInit(): void {
  }

  public async onFileSelected(event: any) {
    const file: File = event.target.files[0];
    this.entrada = await file.text();
  }


  ejecutar(){
    let split_entrada = this.entrada.split("\n");
for (let i = 0; i < split_entrada.length; i++) {
  const cmd = split_entrada[i];
  if (cmd != "") {
    // Buscar un comentario en la línea de comando
    const regex = /([^#]*)#(.*)/;
    const match = regex.exec(cmd);
    let cmdToSend = match ? match[1].trim() : cmd.trim(); // Obtener el comando sin el comentario
    let comment = match ? match[2].trim() : ''; // Obtener el comentario si existe

    this.service.postEntrada(cmdToSend).subscribe(async (res: any) => {
      this.salida += await res.result + "\n";
      if (comment) {
        this.salida += `# Comentario encontrado -> ${comment}\n`; // Agregar el comentario a la salida
      }
    });
  }
}
  }

  // ejecutar() {
  //   this.salida = "--- Resultados ---\n";
  //   let split_entrada = this.entrada.split("\n");
  //   for (let i = 0; i < split_entrada.length; i++) {
  //     const cmd = split_entrada[i];
  //     if (cmd != "") {
  //       this.service.postEntrada(cmd).subscribe(async (res: any) => {
  //         this.salida += await res.result + "\n";
  //       });
  //     }
  //   }
  // }

// ejecutar() {
//   // if (this.entrada === "") {
//   //   Swal.fire({
//   //     title: 'Error',
//   //     text: 'Debe ingresar al menos un comando',
//   //     icon: 'error',
//   //   });
//   //   return;
//   // };
//   this.salida = "--- Resultados ---\n";
//   let split_entrada = this.entrada.split("\n");
//   for (let i = 0; i < split_entrada.length; i++) {
//     const cmd = split_entrada[i];
//     if (cmd != "") {
//       this.service.postEntrada(cmd).subscribe(async (res: any) => {
//         this.salida += await res.result + "\n";
//       });
//     }
//   }
// }

// ejecutar() {
//   // if (!this.entrada.trim()) {
//   //   Swal.fire({
//   //     title: 'Error',
//   //     text: 'Debe llenar todos los campos',
//   //     icon: 'error',
//   //   });
//   //   return;
//   // };
//   this.salida = "--- Resultados ---\n";
//   let split_entrada = this.entrada.split("\n");
//   for (let i = 0; i < split_entrada.length; i++) {
//     const cmd = split_entrada[i];
//     if (cmd != "") {
//       this.service.postEntrada(cmd).subscribe(async (res: any) => {
//         this.salida += await res.result + "\n";
//       });
//     }
//   }
// }

}
