import { Component } from '@angular/core';
import { ProyectoService } from 'src/app/services/proyecto.service';
import { delay } from 'rxjs/operators';
import Swal from 'sweetalert2';


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

  ejecutar() {
    if (!this.entrada) {
      // Si this.entrada está vacío, no hacer nada
      Swal.fire({
        icon: 'error',
        title: 'Oops...',
        text: 'Parece que no hay nada para ejecutar!'
      })
      return;
    }
    let split_entrada = this.entrada.split("\n");
  
    (async () => {
      for (let i = 0; i < split_entrada.length; i++) {
        const cmd = split_entrada[i];
        if (cmd != "") {
          // Buscar un comentario en la línea de comando
          const regex = /([^#]*)#(.*)/;
          const match = regex.exec(cmd);
          let cmdToSend = match ? match[1].trim() : cmd.trim(); // Obtener el comando sin el comentario
          let comment = match ? match[2].trim() : ''; // Obtener el comentario si existe

          if (cmdToSend.includes("rmdisk")){
            let shouldSkip = false; // Establecer una variable de control de salto en falso
            await Swal.fire({
              title: 'Desea eliminar este disco?',
              text: "No podrás recuperar los datos!",
              icon: 'warning',
              showCancelButton: true,
              confirmButtonColor: '#3085d6',
              cancelButtonColor: '#d33',
              cancelButtonText: 'Cancelar!',
              confirmButtonText: 'Si, quiero eliminarlo!'
            }).then((result) => {
              if (result.isConfirmed) {
                Swal.fire(
                  'Se ha enviado la petición de borrar disco!',
                  'El disco será eliminado si se encuentra.',
                  'success'
                )
              } else {
                // Establecer la variable de salto en verdadero
                shouldSkip = true;
                Swal.fire(
                  'Eliminación de disco cancelada!',
                  'El disco no será eliminado.',
                  'info'
                )
              }
            });
            // Si la variable de salto es verdadera, omitir el comando
            if (shouldSkip) {
              continue;
            }
          }

          if (cmdToSend.includes("pause")) {
            // Mostrar un mensaje de pausa
            this.salida += "Comando pause encontrado.\n";
            await new Promise<void>(resolve => {
              Swal.fire({
                title: 'Se ha pausado la ejecucion del proyecto',
                icon: 'info',
                showCancelButton: true,
                confirmButtonText: 'Continuar ejecucion',
                cancelButtonText: 'Detener ejecucion',
                reverseButtons: true
              }).then(result => {
                if (result.isConfirmed) {
                  resolve(); // Resolver la promesa para continuar el proceso
                } else {
                  this.salida += "Se ha cancelado la ejecucion del proyecto.\n";
                }
              });
            });
          } else {
            // ...
            // Esperar antes de enviar la siguiente petición
            await this.service.postEntrada(cmdToSend).pipe(
              delay(500) // Espera de medio segundo
            ).toPromise().then(async (res: any) => {
              this.salida += await res.result + "\n";
              if (comment) {
                this.salida += `# Comentario encontrado -> ${comment}\n`; // Agregar el comentario a la salida
              }
            });
          }
        }
      }
    })();
  }

  
  limpiar(){
    this.entrada = ""
    this.salida = ""
  }

}
